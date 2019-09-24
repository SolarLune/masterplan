go build -o ./bin/MasterPlan ./src/
# Cross-compilation doesn't error out, but doesn't seem to run on Windows (and probably the same for Mac, if I were to try it).
# I only have 64-bit compilers on my computer; TODO: See if I can come back another day to compile in 32-bit mode
# for Windows.
# CGO_ENABLED=1 CC=/usr/bin/cc CXX=/usr/bin/g++ GO_OS=windows go build -o ./bin/MasterPlan.exe ./src/
# CGO_ENABLED=1 CC=/usr/bin/cc CXX=/usr/bin/g++ GO_OS=windows GOARCH=386 go build -o ./bin/MasterPlan.exe ./src/
