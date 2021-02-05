package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var root string = "/Users/pkrishnasamy/Documents/poornima-krishnasamy/cloud-platform-environments/namespaces/cp-2501-1602.cloud-platform.service.justice.gov.uk"

type Results struct {
	Response string
	Error    string
	Folder   string
}

func applyNamespaceDirs(wg *sync.WaitGroup, results chan Results, chunkFolder []string) {
	defer wg.Done()

	for _, folder := range chunkFolder {
		var result Results

		var outb, errb bytes.Buffer

		kubectlArgs := []string{"-n", filepath.Base(folder), "apply", "-f", folder}

		kubectlCommand := exec.Command("kubectl", kubectlArgs...)

		kubectlCommand.Stdout = &outb
		kubectlCommand.Stderr = &errb
		kubectlCommand.Run()

		result = Results{Response: errb.String(), Error: errb.String(), Folder: folder}

		// results <- result

		// Get the value of an Environment Variable
		bucket := os.Getenv("PIPELINE_STATE_BUCKET")
		key_prefix := os.Getenv("PIPELINE_STATE_KEY_PREFIX")
		lock_table := os.Getenv("PIPELINE_TERRAFORM_STATE_LOCK_TABLE")
		region := os.Getenv("PIPELINE_STATE_REGION")
		cluster := os.Getenv("PIPELINE_CLUSTER")

		// //Checking that an environment variable is present or not.

		key := key_prefix + cluster + "/" + filepath.Base(folder) + "/terraform.tfstate"

		// // err := os.RemoveAll(filepath.Join(folder, ".terraform"))
		// // if err != nil {
		// // 	result = Result{Error: "Cant remove .terraform folders", Response: "", Folder: folder}
		// // 	results <- result
		// // 	return
		// // }

		kubectlArgs = []string{
			"init",
			fmt.Sprintf("%s=bucket=%s", "-backend-config", bucket),
			fmt.Sprintf("%s=key=%s", "-backend-config", key),
			fmt.Sprintf("%s=dynamodb_table=%s", "-backend-config", lock_table),
			fmt.Sprintf("%s=region=%s", "-backend-config", region)}

		Command := exec.Command("terraform", kubectlArgs...)

		Command.Dir = folder + "/resources"
		Command.Stdout = &outb
		Command.Stderr = &errb
		Command.Run()

		kubectlArgs = []string{"plan"}

		Command = exec.Command("terraform", kubectlArgs...)

		Command.Dir = folder + "/resources"
		Command.Stdout = &outb
		Command.Stderr = &errb
		Command.Run()

		kubectlArgs = []string{"apply"}

		Command = exec.Command("terraform", kubectlArgs...)

		Command.Dir = folder + "/resources"
		Command.Stdout = &outb
		Command.Stderr = &errb
		Command.Run()

		result = Results{Response: outb.String(), Error: errb.String(), Folder: folder}

		results <- result
	}

}

func monitorResults(wg *sync.WaitGroup, results chan Results) {
	wg.Wait()
	close(results)
}

func listFiles(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() {
			fmt.Printf("skipping a dir without errors: %+v \n", info.Name())
			return filepath.SkipDir
		}
		*files = append(*files, path)
		return nil
	}
}

func listFolders(folders *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.Name() == ".terraform" || info.Name() == "resources" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			*folders = append(*folders, path)
		}
		return nil
	}
}

func main() {

	fmt.Printf("START TIME %s \n", time.Now().String())

	fmt.Println("Version", runtime.Version())
	fmt.Println("NumCPU", runtime.NumCPU())
	fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))

	const nRoutines int = 4

	env_vars := [3]string{
		"TF_VAR_cluster_name",
		"TF_VAR_cluster_state_bucket",
		"TF_VAR_cluster_state_key"}

	for _, env_var := range env_vars {

		// `os.Getenv` cannot differentiate between an explicitly set empty value
		// and an unset value. `os.LookupEnv` is preferred to `syscall.Getenv`,
		// but it is only available in go1.5 or newer. We're using Go build tags
		// here to use os.LookupEnv for >=go1.5
		_, ok := os.LookupEnv(env_var)
		if !ok {
			panic(ok)
		}

		continue
	}

	var folders []string

	err := filepath.Walk(root, listFolders(&folders))
	if err != nil {
		panic(err)
	}

	fmt.Println("Number of folders", len(folders))

	nChunks := len(folders) / nRoutines

	fmt.Println("Number of folders per chunk", nChunks)

	var folderChunks [][]string
	for {
		if len(folders) == 0 {
			break
		}

		if len(folders) < nChunks {
			nChunks = len(folders)
		}

		folderChunks = append(folderChunks, folders[0:nChunks])
		folders = folders[nChunks:]
	}

	results := make(chan Results)

	wg := &sync.WaitGroup{}

	// TODO: create a pool of threads and spread the folders to the given threads. This is because
	// The number of max threads which can call the AWS api should be limited to avoid exceeding the rate limits

	fmt.Println("Number of Chunks", len(folderChunks))
	for i := 0; i < len(folderChunks); i++ {
		wg.Add(1)
		go applyNamespaceDirs(wg, results, folderChunks[i])

	}

	go monitorResults(wg, results)

	for result := range results {
		fmt.Printf("Folder: %v\n", result.Folder)
		fmt.Printf("Response: %v\n", result.Response)
		if result.Error != "" {
			fmt.Printf("Error: %v", result.Error)
			continue
		}
	}

	fmt.Printf("END TIME %s \n", time.Now().String())

}
