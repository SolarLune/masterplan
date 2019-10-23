go build -o ./bin/MasterPlan ./src/

# Building on one's own system using the above command works fine, but cross-compilation doesn't work with vanilla Go
# because I'm using raylib-go and CGO (C + Go) doesn't work with cross-compilation. So cross-compilation might be able to
# work using techknowlogik's xgo fork.

# sudo dockerd  # Run this in another terminal

# gox -cgo -osarch="linux/amd64 windows/386" -output="bin/{{.OS}}_{{.Arch}}" ./src/
 
# sudo /home/solarlune/Documents/Projects/Go/bin/xgo -out MasterPlan --targets=linux/amd64 github.com/SolarLune/masterplan

# CGO_ENABLED=1 CC_FOR_TARGET=gcc GOOS=windows GOARCH=amd64 go build -o ./bin/MasterPlan.exe ./src/
