package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

var wg sync.WaitGroup

func main() {
	user := User{}
	err := user.Configurate()
	if err != nil {
		log.Fatal(err)
	}

	isRDS := flag.Bool("rds", false, "Sync repos on RDS")
	flag.Parse()

	client := &http.Client{}
	repos, err := getForkedRepos(user.Token, client)
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(len(repos))
	for _, v := range repos {
		v := v
		go func() {
			defer wg.Done()
			err := fetchUpstream(user.Username, user.Token, v, client)
			if err != nil {
				if err.Error() == "404" {
					return
				}
				log.Println(err)
				return
			}
			err = syncLocally(user.ReposLocation, v)
			if err != nil {
				log.Println(err)
				return
			} else {
				if *isRDS {
					err := syncRDS(user.ReposLocation, v)
					if err != nil {
						log.Println(err)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

func getForkedRepos(token string, client *http.Client) ([]string, error) {
	var url = string("https://api.github.com/user/repos?visibility=private&affiliation=organization_member&per_page=100")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	by, _ :=ioutil.ReadAll(resp.Body)

	var result []map[string]string
	json.Unmarshal([]byte(by), &result)

	var repos []string
	for i := 0; i < len(result); i++ {
		repos = append(repos, result[i]["name"])
	}
	return repos, nil
}

func fetchUpstream(username, token, repoName string, client *http.Client) error {
	var branch = []byte(`{"branch":"master"}`)
	var url = string(fmt.Sprintf("https://api.github.com/repos/%s/%s/merge-upstream", username, repoName))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(branch))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("%s successfully fetched upstream!\n", repoName)
		return nil
	} else if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("there are some conflicts in %s", repoName)
	} else if resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("unprocessable entity %s", repoName)
	} else {
		return fmt.Errorf("%d", resp.StatusCode)
	}
}

func syncLocally(location, repo string) error {
	dir := fmt.Sprintf("%s/%s", location, repo)
	swt := exec.Command("git", "switch", "master")
	swt.Dir = dir
	err := swt.Run()
	if err != nil {
		return err
	}

	rebase := exec.Command("git", "pull", "--rebase")
	rebase.Dir = dir
	err = rebase.Run()
	if err != nil {
		return err
	}
	log.Printf("%s locally synced 🥳\n", repo)
	return nil
}

func syncRDS(location, repo string) error {
	dir := fmt.Sprintf("%s/%s", location, repo)
	sa := exec.Command("./syncAll")
	sa.Dir = dir
	err := sa.Run()
	if err != nil {
		return err
	} 
	log.Printf("%s synced on your RDS 🥸", repo)
	return nil
}