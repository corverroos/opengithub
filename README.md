# Opengithub

> `opengithub` is a simple tool to open source code in Github. 

It can also be used to quickly and easily share links to locally checked-out code.

## Usage 

1. Install `opengithub`:
```shell
go install github.com/corverroos/opengithub
```

2. Copy a reference to local checked-out source code to the clipboard:
   - IntelliJ on MacOS, select a line of code and either press `ALT-SHIFT-COMMAND-C` or `Right-click > Copy > Copy Reference`.
   - The copied code should be in the format: `path/to/file.ext` or `path/to/file.ext:123`

3. Run `opengithub` on your cli to open that file in Github using the default browser:
```shell
opengithub
```

4. Optionally configure `opengithub`:
   - Explicitly provide the file (default to clipboard): `--file=repo/folder/file.go:123`
   - Disable auto opening: `--open=false`
   - Root directory to search for resolve paths (defaults to current dir): `--root=/path/to/my/repos` or `export OPENGITHUB_ROOT=/path/to/my/repos`
   - Git branch to use (defaults to current branch): `--branch=main` or `export OPENGITHUB_BRANCH=main`
   - Concise alias: `alias ogh="opengithub --root=/path/to/my/repos --branch=main"`

5. Configure it as an Intellij "External Tool":
   - `Preferences > Tools > External Tools > Add`
     - Program: `opengithub`
     - Arguments: `--file=$FilePath$:$LineNumber$`
     - Working Directory: `$ProjectFileDir$`
   - Then just select some code, then select `Tools > External Tools > opengithub`
   
![Add IntelliJ External Tool](intellij.png "Add IntelliJ External Tool")

## Notes

This is similar to the IntelliJ plugin [Open in Github](https://plugins.jetbrains.com/plugin/7190-open-in-github) which I couldn't get to work.