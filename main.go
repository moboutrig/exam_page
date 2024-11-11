package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve code and arguments from the form submission
	mainCode := r.FormValue("main")
	funcCode := r.FormValue("func")
	args := r.FormValue("args")

	// Replace "piscine" with "compiler_task/piscine" in mainCode
	mainCode = strings.Replace(mainCode, `"piscine"`, `"compiler_task/piscine"`, 1)

	// Temporary directory to store the code files
	tmpDir := "/tmp/compiler_task"
	err := os.MkdirAll(tmpDir+"/piscine", 0755)
	if err != nil {
		http.Error(w, "Failed to create directories", http.StatusInternalServerError)
		return
	}

	// Check if go.mod exists, and only initialize if it doesn't
	goModPath := fmt.Sprintf("%s/go.mod", tmpDir)
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// Initialize a new Go module in the temporary directory
		modCmd := exec.Command("go", "mod", "init", "compiler_task")
		modCmd.Dir = tmpDir
		var modOut bytes.Buffer
		modCmd.Stdout = &modOut
		modCmd.Stderr = &modOut
		err = modCmd.Run()
		if err != nil {
			http.Error(w, "Failed to initialize Go module: "+modOut.String(), http.StatusInternalServerError)
			return
		}
	}

	// Write the func code to a file
	funcFile := fmt.Sprintf("%s/piscine/func.go", tmpDir)
	err = os.WriteFile(funcFile, []byte(funcCode), 0644)
	if err != nil {
		http.Error(w, "Failed to write func.go", http.StatusInternalServerError)
		return
	}

	// Write the main code to a file
	mainFile := fmt.Sprintf("%s/main.go", tmpDir)
	err = os.WriteFile(mainFile, []byte(mainCode), 0644)
	if err != nil {
		http.Error(w, "Failed to write main.go", http.StatusInternalServerError)
		return
	}

	// Fetch the required module
	fetchCmd := exec.Command("go", "get", "github.com/01-edu/z01")
	fetchCmd.Dir = tmpDir
	var fetchOut bytes.Buffer
	fetchCmd.Stdout = &fetchOut
	fetchCmd.Stderr = &fetchOut
	err = fetchCmd.Run()
	if err != nil {
		http.Error(w, "Failed to fetch required module: "+fetchOut.String(), http.StatusInternalServerError)
		return
	}

	// Build the Go code
	buildCmd := exec.Command("go", "build", "-o", "program", "main.go")
	buildCmd.Dir = tmpDir
	var buildOut bytes.Buffer
	buildCmd.Stdout = &buildOut
	buildCmd.Stderr = &buildOut
	err = buildCmd.Run()
	if err != nil {
		http.Error(w, "Failed to build Go code: "+buildOut.String(), http.StatusInternalServerError)
		return
	}

	// Split the args string by whitespace to pass as individual arguments
	runArgs := strings.Fields(args)
	runCmdArgs := append([]string{fmt.Sprintf("%s/program", tmpDir)}, runArgs...)

	// Run the compiled executable
	runCmd := exec.Command(runCmdArgs[0], runCmdArgs[1:]...)
	var runOut bytes.Buffer
	runCmd.Stdout = &runOut
	runCmd.Stderr = &runOut
	err = runCmd.Run()
	if err != nil {
		runOut.WriteString("\nError: " + err.Error())
	}

	// Clean up the program file
	os.Remove(fmt.Sprintf("%s/program", tmpDir))

	// Send the output back to the client
	w.Write([]byte(runOut.String()))
}

func main() {
	// Serve static files (HTML, CSS, JS) from the "static" directory
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve the main page (index.html) at the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html") // Ensure this points to your actual index.html
	})

	// Handle the compilation and execution of the Go code
	http.HandleFunc("/compile", handleCompile)
	http.HandleFunc("/test", handletest)
	// Start the server
	fmt.Println("Server started at http://localhost:3429")
	err := http.ListenAndServe(":3429", nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func handletest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve code and arguments from the form submission
	funcCode := r.FormValue("func")
    mainCode := r.FormValue("main")
	subject := r.FormValue("subject")
   
    fmt.Println( mainCode , subject , funcCode)
    
	if strings.HasSuffix(subject, ".go") {
		subject = subject[:len(subject)-3] // Remove ".go" from the subject name
	}
	// Define the paths for the subject files using the subject identifier
	subjectFilePath := fmt.Sprintf("./go-tests/piscine-go/%s.go", subject)
	mainFilePath := fmt.Sprintf("./go-tests/piscine-go/%s/main.go", subject)
	fmt.Println(subjectFilePath, mainFilePath)
	// Check if main.go exists and use mainCode if so

	if _, err := os.Stat(mainFilePath); err == nil {
		// If main.go exists for the subject, write the mainCode to it
		err := ioutil.WriteFile(mainFilePath, []byte(mainCode), 0644)
		if err != nil {
			http.Error(w, "Error writing to main.go", http.StatusInternalServerError)
			return
		}
	} else if _, err := os.Stat(subjectFilePath); err == nil {
		// If subject.go exists, write the funcCode to it
		err := ioutil.WriteFile(subjectFilePath, []byte(funcCode), 0644)
		if err != nil {
			http.Error(w, "Error writing to subject.go", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Subject file not found", http.StatusNotFound)
		return
	}

	// Run the test script with the subject as a parameter
cmd := exec.Command("./test_one.sh", subject)
cmd.Dir = "./go-tests" // Set working directory to go-tests folder

output, err := cmd.CombinedOutput()
if err != nil {
    // If there's an error, return the error output
    w.WriteHeader(http.StatusInternalServerError) // Set the correct HTTP status
    w.Write([]byte(fmt.Sprintf("%v\n%s", err, string(output))))
    return
}

	// If output is empty, return the correct solution message
	if strings.TrimSpace(string(output)) == "" {
		w.Write([]byte("Correct solution"))
	} else {
		w.Write(output) // Return the output from the script
	}
}
