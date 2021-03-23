# dllogram

## Building:

### 64 bit

```
x86_64-w64-mingw32-g++ -o nice.exe build/*

x86_64-w64-mingw32-g++ -shared -o msvcp140.dll build/*
```

### 32 bit

```
i686-w64-mingw32-g++ -o nice.exe build/*

i686-w64-mingw32-g++ -shared -o msvcp140.dll build/*
```


