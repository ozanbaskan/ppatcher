## Build

You need to have Tkinter installed on your system to build it, for linux you can use install it by `apt-get install python3-tk`.

```bash
# Testing

# Create the manifest.json
python manifest.py

# Build the program
pyinstaller --onefile --noconsole --clean patcher.py

# Enter the directory since the patcher will download to the current directory
cd dist

# Run the patcher
./patcher
```

If you need something simple, to use it as it is, run the manifest.py from your server side and serve the files including the manifest (the directories are specified in the config.py), then you can build the client side or use it, simply add the builded executable to your files' root directory.