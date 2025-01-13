$COMMAND = $args[0]

# Set constants
$NAME = "qsc-control"
$OWNER = "byuoitav"
$PKG = "github.com/$OWNER/$NAME"
$DOCKER_URL = "ghcr.io"
$DOCKER_PKG = "$DOCKER_URL/$OWNER/$NAME"

Write-Output "PKG: $PKG"
Write-Output "DOCKER_PKG: $DOCKER_PKG"

# Regular expressions for version tags
$PRD_TAG_REGEX = "v[0-9]+\.[0-9]+\.[0-9]+"
$DEV_TAG_REGEX = "v[0-9]+\.[0-9]+\.[0-9]+-.+"

# Get commit hash and tag
$COMMIT_HASH = Invoke-Expression "git rev-parse --short HEAD"
$TAG = Invoke-Expression "git rev-parse --short HEAD"
try {
    $NEW_TAG = Invoke-Expression "git describe --exact-match --tags HEAD"
    Write-Output "NEW_TAG: $NEW_TAG.Length"
    if ($NEW_TAG.Length -gt 0) {
        $TAG = $NEW_TAG
        Write-Output "The repo contains a tag: $TAG"
    }
}
catch {
    Write-Output "The repo does not contain a tag"
}

Write-Output "The TAG is: $TAG"

# Go package list
$PKG_LIST = Invoke-Expression "go list $PKG/..."
Write-Output "PKG_LIST: $PKG_LIST"

function All {
    Write-Output "Running All"
    Cleanup
    Build
}

function Test {
    Write-Output "Running Test"
    Invoke-Expression "go test -v $PKG_LIST"
}

function Test-cov {
    Write-Output "Running Test-cov"
    Invoke-Expression "go test -coverprofile=coverage.txt -covermode=atomic $PKG_LIST"
}

function Lint {
    Write-Output "Running Lint"
    Invoke-Expression "golangci-lint run --tests=false"
}

function Deps {
    Write-Output "Downloading Dependencies"
    Invoke-Expression "go mod download"
}

function Build {
    Write-Output "Building project"
    New-Item -Path dist -ItemType Directory -Force

    Set-Location "cmd"
    Write-Output "Building for linux-amd64"
    Set-Item -Path env:CGO_ENABLED -Value 0
    Set-Item -Path env:GOOS -Value "linux"
    Set-Item -Path env:GOARCH -Value "amd64"
    Invoke-Expression "go build -o ../dist/$NAME-linux-amd64"
    
    Write-Output "Building for linux-arm"
    Set-Item -Path env:GOARCH -Value "arm"
    Invoke-Expression "go build -o ../dist/$NAME-linux-arm"

    Write-Output "Build output is located in ./dist/."
    Set-Item -Path env:GOOS -Value "windows"
    Set-Item -Path env:GOARCH -Value "amd64"
    Set-Location ".."
}

function Cleanup {
    Write-Output "Cleaning project"
    Invoke-Expression "go clean"
    if (Test-Path -Path "dist") {
        Remove-Item dist -Recurse -Force
        Write-Output "Deleted dist/ directory"
    } else {
        Write-Output "No dist directory to delete"
    }
}

function DockerFunc {
    Write-Output "Building Docker images for Commit Hash: $COMMIT_HASH, Tag: $TAG"
    if ($COMMIT_HASH -eq $TAG) {
        Write-Output "Building dev containers with tag $COMMIT_HASH"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-amd64 -t $DOCKER_PKG/$NAME-dev:$COMMIT_HASH dist"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-arm -t $DOCKER_PKG/$NAME-arm-dev:$COMMIT_HASH dist"
    } elseif ($TAG -match $DEV_TAG_REGEX) {
        Write-Output "Building dev containers with tag $TAG"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-amd64 -t $DOCKER_PKG/$NAME-dev:$TAG dist"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-arm -t $DOCKER_PKG/$NAME-arm-dev:$TAG dist"
    } elseif ($TAG -match $PRD_TAG_REGEX) {
        Write-Output "Building prod containers with tag $TAG"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-amd64 -t $DOCKER_PKG/${NAME}:$TAG dist"
        Invoke-Expression "docker buildx build -f .\dockerfile --platform linux/arm/v7  --build-arg NAME=$NAME-linux-arm -t $DOCKER_PKG/$NAME-arm:$TAG dist"
    } else {
        Write-Output "Unexpected state. Commit Hash: $COMMIT_HASH, Tag: $TAG"
    }
}

function Deploy {
    Write-Output "Deploying to Docker with Commit Hash: $COMMIT_HASH, Tag: $TAG"
    Invoke-Expression "docker login $DOCKER_URL -u $Env:DOCKER_USERNAME -p $Env:DOCKER_PASSWORD"
    
    if ($COMMIT_HASH -eq $TAG) {
        Write-Output "Pushing dev containers with tag $COMMIT_HASH"
        Invoke-Expression "docker push $DOCKER_PKG/$NAME-dev:$COMMIT_HASH"
        Invoke-Expression "docker push $DOCKER_PKG/$NAME-arm-dev:$COMMIT_HASH"
    } elseif ($TAG -match $DEV_TAG_REGEX) {
        Write-Output "Pushing dev containers with tag $TAG"
        Invoke-Expression "docker push $DOCKER_PKG/$NAME-dev:$TAG"
        Invoke-Expression "docker push $DOCKER_PKG/$NAME-arm-dev:$TAG"
    } elseif ($TAG -match $PRD_TAG_REGEX) {
        Write-Output "Pushing prod containers with tag $TAG"
        Invoke-Expression "docker push $DOCKER_PKG/${NAME}:$TAG"
        Invoke-Expression "docker push $DOCKER_PKG/$NAME-arm:$TAG"
    } else {
        Write-Output "Unexpected state. Commit Hash: $COMMIT_HASH, Tag: $TAG"
    }
}

# Main execution based on command argument
if ($COMMAND -eq "All") {
    Cleanup
    Build
    All
}
elseif ($COMMAND -eq "Test") {
    Deps
    Test
}
elseif ($COMMAND -eq "Test-cov") {
    Deps
    Test-cov
}
elseif ($COMMAND -eq "Lint") {
    Deps
    Lint
}
elseif ($COMMAND -eq "Deps") {
    Deps
}
elseif ($COMMAND -eq "Build") {
    Cleanup
    Deps
    Build
}
elseif ($COMMAND -eq "Clean") {
    Cleanup
}
elseif ($COMMAND -eq "Docker") {
    Cleanup
    Deps
    Build
    DockerFunc
    Cleanup
}
elseif ($COMMAND -eq "Deploy") {
    Cleanup
    Deps
    Build
    DockerFunc
    Deploy
    Cleanup
}
else {
    Write-Output "Please provide a valid command parameter."
}
