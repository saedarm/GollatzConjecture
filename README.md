# GollatzConjecture

1) Install Golang from https://go.dev/dl/


2) Open Git Bash or CMD

mkdir yourgame cd yourgame go mod init foo # or github.com/yourname/yourgame or something

cd yourgame

Enter "code ." within terminal to open vs code (Or open created directory in any ide)

3) Copy collatgraph.go file from github into new go file in your directory that you can name anything you'd like

4) Run command : Go Get github.com/hajimehoshi/ebiten/v2

5) These are all the imports you'll need
**
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "golang.org/x/image/font"
    "golang.org/x/image/font/opentype"
    "golang.org/x/image/font/gofont/goregular"
    "github.com/hajimehoshi/ebiten/v2/text"**

6) Run command : Go mod tidy

7) Run Command : Go build "name_of_main_file.go"

8) Run Command : Go run "name_of_main_file.go"



9) Enter Integer in field greater than 0 ( no negative numbers) and stopping time will be rendered in an ebiten window with vertices or nodes representing each iteration that the algorithm takes
