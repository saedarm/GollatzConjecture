# Project Title

A brief description of what this project does and who it's for
Gollatz Conjecture - Collatz Visualizer with Golang and Ebiten

# Step 1 : Installation
Install Golang from https://go.dev/dl/

# Step 2 : Git Bash/CMD Init

mkdir yourgame cd yourgame go mod init foo # or github.com/yourname/yourgame or something

cd yourgame

Enter "code ." within terminal to open vs code (Or open created directory in any ide)

# Step 3: Copy Code

Copy collatgraph.go file from github into new go file in your directory that you can name anything you'd like

# Step 4: Go Get Command and Imports

Run command : Go Get github.com/hajimehoshi/ebiten/v2

These are all the imports you'll need

** "github.com/hajimehoshi/ebiten/v2" "github.com/hajimehoshi/ebiten/v2/vector" "github.com/hajimehoshi/ebiten/v2/inpututil" "golang.org/x/image/font" "golang.org/x/image/font/opentype" "golang.org/x/image/font/gofont/goregular" "github.com/hajimehoshi/ebiten/v2/text"**
# Step 5 : Build and Cleanup
Run command : Go mod tidy

Run Command : Go build "name_of_main_file.go"

# Step 6 : Run the Program
Run Command : Go run "name_of_main_file.go"

Enter Integer in field greater than 0 ( no negative numbers) and stopping time will be rendered in an ebiten window with vertices or nodes representing each iteration that the algorithm takes

