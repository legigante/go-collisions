package main

import (
	"fmt"
	"errors"
	"net/http"
	"bufio"
	"strings"
	"regexp"
	"encoding/json"
	"time"
)

// json format {fileName: nbCollisions}
type structChan struct {
	fileName string
	nbCollisions int
}

var G_regexUtf8 = "^[A-Za-zÀÁÂÃÄÅàáâãäåÒÓÔÕÖØòóôõöøÈÉÊËèéêëÇçÌÍÎÏìíîïÙÚÛÜùúûüÿÑñ]+$"






// input data
var listUrl = "https://gist.githubusercontent.com/alexcesaro/c9c47c638252e21bd82c/raw/bd031237a56ae6691145b4df5617c385dffe930d/list.txt"

var filesPath = [4]string{
	"https://gist.githubusercontent.com/alexcesaro/4ebfa5a9548d053dddb2/raw/abb8525774b63f342e5173d1af89e47a7a39cd2d/file1.txt",
	"https://gist.githubusercontent.com/alexcesaro/249cde1332f9b2979140/raw/951e43186f14f9c386918d75d715ee49390ebc54/file2.txt",
	"https://gist.githubusercontent.com/alexcesaro/f99d72a1d1a1f140b27f/raw/e506ed86336ea8561027d9f8cd4007d1f691d835/file3.txt",
	"https://gist.githubusercontent.com/alexcesaro/77c12bfd58a0d1156d77/raw/e40a381a4bfade72c8fe85e97d83736a48f091e6/file4.txt",
}




func main() {
    http.HandleFunc("/", handler)
	fmt.Println("Listening port 8080...")
    http.ListenAndServe(":8080", nil)
}




// handler routs and serve a response to a request
// ***TODO V2: if more pages => do a MVC archi + rout to controller => views handled by html/template
func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[1:] == "" {
		fmt.Println("Call goCollisions")
		res, err := goCollisions()
		if err == nil {
			fmt.Println("goCollisions succeed")
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, res)
		} else {
			fmt.Println("goCollisions failed: " + err.Error())
			http.Error(w, err.Error(), 500)
		}
	} else {
		fmt.Println("404 error: " + r.URL.Path[1:])
		http.Error(w, res404(), 404)
	}
}




// res404 generates 404 page
func res404() string{
	return "This url is not valid :("
}




// This program compares a list of words with several other list contained in several files.
// The program returns the number of collisions for each file in a json format {"file name", "number of collisions"}
func goCollisions() (string, error) {
	ch := make(chan structChan)
	mapJson := make(map[string]int)

	// get list input
	mapList, err := getListMap(listUrl)
	if err != nil {
		// list not valid: critic error
		return "", err
	}

	// compute each file
	for _, el := range filesPath {
		go func(el string) {
			filePath, err := getFileNameFromPath(el)
			if err == nil {
				n, err := getNbUnionListVsFile(mapList, el)
				if err == nil {
					ch <- structChan{filePath, n}
				} else {
					// file content not vaid: warning + display -1 to user 
					// ***TODO V2: create a log of not valid lines to show to the user
					fmt.Println(err)
					ch <- structChan{filePath, -1}
				}
			} else {
				// file name not valid: warning error
				fmt.Println(err)
			}
		}(el)
	}
	
	// wait for channel
	for {
		select {
		case r := <-ch:
			mapJson[r.fileName] = r.nbCollisions
			if len(mapJson) == len(filesPath) {
				strJsonOutput, err := json.Marshal(mapJson)
				if err != nil {
					// json not valid: critic error
					return "", err
				}
				return string(strJsonOutput), nil
			}

		case <-time.After(10000 * time.Millisecond):
			// timeout: critic error
			return "", errors.New("Timeout: operation aborted")
		}
	}
}




// getListMap takes as an input a url.
// Url content format must be one word per line.
// getListMap returns a map[string]bool containing {key: one word, value: true}.
// Note that duplicates are killed by this process
func getListMap(strListPath string) (map[string]bool, error) {
	mapReturn := make(map[string]bool)

	res, err := http.Get(strListPath)
	if err != nil || res.StatusCode != 200 {
		return mapReturn, errors.New(strListPath + " : this url is not valid")
	}
	scanner := bufio.NewScanner(res.Body)
	
	for scanner.Scan() {
		strWord := scanner.Text()
		// assuming utf-8
		if ok, _ := regexp.MatchString(G_regexUtf8, strWord); ok {
			mapReturn[scanner.Text()] = true
		} else {
			return mapReturn, errors.New(strWord + " : this line is not valid in list")
		}
	}
	res.Body.Close()
	return mapReturn, nil
}




// getUnionListVsFile first argument is a map {key: one word, item: bool}.
// Second argument is a url.
// Url content format must be one word per line. getNbUnionListVsFile returns -1 is the format is not valid.
// getUnionListVsFile counts the number of words present in both map and url content (duplicate are valued once)
func getNbUnionListVsFile(mapList map[string]bool, strFilePath string) (int, error) {
	// mémoire pour gestion des doublons
	mapDuplicate := make(map[string]bool)
	// compteur
	intReturn := 0
	
	res, err := http.Get(strFilePath)
	if err != nil || res.StatusCode != 200 {
		return -1, errors.New(strFilePath + " : this url is not valid")
	}
	scanner := bufio.NewScanner(res.Body)

	for scanner.Scan() {
		strWord := scanner.Text()
		// assuming we're speaking english or french
		if ok, _ := regexp.MatchString(G_regexUtf8, strWord); ok {
			if _, ok := mapDuplicate[strWord]; !ok {
				if _, ok := mapList[strWord]; ok {
					intReturn += 1
				}
			}
		} else {
			return -1, errors.New(strWord + " : this line is not valid in " + strFilePath)
		}
	}
	return intReturn, nil
}




// getFileNameFromPath returns a file name from a complete path name
func getFileNameFromPath(strPath string) (string, error) {
	lim := strings.LastIndex(strPath, "/")
	if lim < 1 {
		return "", errors.New(strPath + " : cannot find file name of this path")
	}
	return strPath[strings.LastIndex(strPath, "/")+1 : len(strPath)], nil
}
