![Typstify Logo]("./appicon.png")

# Typstify

The cross-platform desktop editor for Typst.


## Build

=== windows

==== Cross-compile in MacOS

Install mingw64 first:

```
brew install mingw-w64
```

Then run the following command:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ make windows
```

The CC and CXX are needed by webview_go, Gio is not required to set them.

==== Compile in Windows

Install MinGW-w64 first follow the guides below:

https://github.com/niXman/mingw-builds-binaries?tab=readme-ov-file


and then in git bash:

```
 export PATH="C:\Users\atzha\mingw64\bin:$PATH"
```

You are done.