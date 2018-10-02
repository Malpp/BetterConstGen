package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"hash/fnv"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var projectDirectoryPath string
var outputDirectoryPath string

type ConstClass struct {
	Name    string
	Members []ConstMember
}

type ConstMember struct {
	Id      uint32
	Name    string
	Path    string
	IsValid bool
}

type TagManagerAssetYaml struct {
	TagManager struct {
		Tags   []string `yaml:"tags"`
		Layers []string `yaml:"layers"`
	} `yaml:"TagManager"`
}

type FolderItem struct {
	Name string
	Path string
}

type AnimationControllerYaml struct {
	AnimatorController struct {
		MAnimatorParameters []struct {
			MName string `yaml:"m_Name"`
		} `yaml:"m_AnimatorParameters"`
	} `yaml:"AnimatorController"`
}

type GameObjectYaml struct {
	GameObject struct {
		MName string `yaml:"m_Name"`
	} `yaml:"GameObject"`
}

var constClasses []ConstClass
var sceneItems []FolderItem
var prefabItems []FolderItem
var animationItems []FolderItem

func main() {
	validateArgs()

	projectDirectoryPath = os.Args[1]
	outputDirectoryPath = os.Args[2]

	generateOutputDirectory()

	fmt.Println("Reading files")
	addConstMembersFromTagAndLayer()

	prepareSceneAndPrefabItems()

	addSceneToConstMembers()
	addPrefabToConstMembers()
	addAnimationParametersToConstMembers()
	addGameObjectsToConstMembers()

	generateTemplate()
}

func addGameObjectsToConstMembers() {
	fmt.Println("	GameObjects...")

	var constMembers []ConstMember
	itemsToSearch := append(sceneItems, prefabItems...)

	for _, controllerItem := range itemsToSearch {
		gameObjectYaml := GameObjectYaml{}
		if controllerItem.Path == "" {
			continue
		}
		yamlFile, err := ioutil.ReadFile(controllerItem.Path)
		if err != nil {
			log.Printf("Failed to get file TagManager.asset   #%v ", err)
		}

		yamlFile = bytes.Replace(yamlFile, []byte("\r"), []byte(""), -1)
		allLines := bytes.Split(yamlFile, []byte("\n"))

		var parsedLines string
		var validGameObjects []string
		for _, line := range allLines {
			if strings.HasPrefix(string(line), "--- !u!") {
				if strings.HasPrefix(parsedLines, "GameObject:") {
					validGameObjects = append(validGameObjects, parsedLines)
				}
				parsedLines = ""
				continue
			}
			parsedLines += string(line) + "\n"
		}

		for _, gameObject := range validGameObjects {
			err = yaml.Unmarshal([]byte(gameObject), &gameObjectYaml)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
			constMembers = append(constMembers, createConstMember(gameObjectYaml.GameObject.MName, controllerItem.Path))
		}

	}
	constMembers = append(constMembers, createConstMember("None", ""))
	constClasses = append(constClasses, ConstClass{Name: "GameObject", Members: constMembers})
}

func addAnimationParametersToConstMembers() {
	fmt.Println("	Animation Parameters...")

	var constMembers []ConstMember

	for _, controllerItem := range animationItems {
		animationControllerYaml := AnimationControllerYaml{}
		yamlFile, err := ioutil.ReadFile(controllerItem.Path)
		if err != nil {
			log.Printf("Failed to get file TagManager.asset   #%v ", err)
		}

		err = yaml.Unmarshal(yamlFile, &animationControllerYaml)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		for _, parameter := range animationControllerYaml.AnimatorController.MAnimatorParameters {
			constMembers = append(constMembers, createConstMember(parameter.MName, controllerItem.Path))
		}
	}
	constMembers = append(constMembers, createConstMember("None", ""))
	constClasses = append(constClasses, ConstClass{Name: "AnimationParameters", Members: constMembers})
}

func prepareSceneAndPrefabItems() {
	err := filepath.Walk(path.Join(projectDirectoryPath, "Assets"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", projectDirectoryPath, err)
			return err
		}

		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), ".unity") {
				sceneItems = append(sceneItems, FolderItem{Name: strings.Split(info.Name(), ".")[0], Path: path})
			}

			if strings.HasSuffix(info.Name(), ".prefab") {
				prefabItems = append(prefabItems, FolderItem{Name: strings.Split(info.Name(), ".")[0], Path: path})
			}

			if strings.HasSuffix(info.Name(), ".controller") {
				animationItems = append(animationItems, FolderItem{Name: strings.Split(info.Name(), ".")[0], Path: path})
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", projectDirectoryPath, err)
	}
}
func addSceneToConstMembers() {
	fmt.Println("	Scenes...")
	sceneItems = append(sceneItems, FolderItem{Name: "None", Path: ""})
	constClasses = append(constClasses, ConstClass{Name: "Scene", Members: folderItemsToConstMembers(sceneItems)})
}

func addPrefabToConstMembers() {
	fmt.Println("	Prefabs...")
	prefabItems = append(prefabItems, FolderItem{Name: "None", Path: ""})
	constClasses = append(constClasses, ConstClass{Name: "Prefab", Members: folderItemsToConstMembers(prefabItems)})
}

func folderItemsToConstMembers(items []FolderItem) []ConstMember {
	var constMembers []ConstMember

	for _, item := range items {
		constMembers = append(constMembers, createConstMember(item.Name, item.Path))
	}

	return constMembers
}

func addConstMembersFromTagAndLayer() {
	fmt.Println("	Tags & Layers...")

	manager := TagManagerAssetYaml{}
	tagManagerPath := path.Join(projectDirectoryPath, "ProjectSettings", "TagManager.asset")

	yamlFile, err := ioutil.ReadFile(tagManagerPath)
	if err != nil {
		log.Printf("Failed to get file TagManager.asset   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, &manager)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	addTagConstMembers(manager.TagManager.Tags, tagManagerPath)
	addLayerConstMembers(manager.TagManager.Layers, tagManagerPath)
}

func addTagConstMembers(tags []string, tagManagerPath string) {
	tags = append(tags, "None")
	tags = append(tags, "Untagged")
	tags = append(tags, "Respawn")
	tags = append(tags, "Finish")
	tags = append(tags, "EditorOnly")
	tags = append(tags, "Player")
	tags = append(tags, "MainCamera")
	addConstClassFromMembers("Tags", tags, tagManagerPath)
}

func addLayerConstMembers(layers []string, tagManagerPath string) {
	layers = append(layers, "None")
	addConstClassFromMembers("Layers", layers, tagManagerPath)
}

func addConstClassFromMembers(name string, members []string, path string) {
	constMembers := createMultiConstMember(members, path)
	constClasses = append(constClasses, ConstClass{Name: name, Members: constMembers})
}

func createMultiConstMember(names []string, path string) []ConstMember {
	var constMembers []ConstMember

	for _, name := range removeNilFrom(names) {
		constMembers = append(constMembers, createConstMember(name, path))
	}

	return constMembers
}

func generateTemplate() {
	fmt.Println("\nGenerating C#")

	t, err := template.ParseFiles("R.cs.template")
	if err != nil {
		log.Print(err)
		return
	}

	f, err := os.Create(path.Join(outputDirectoryPath, "R.cs"))
	if err != nil {
		log.Println("create file: ", err)
		return
	}

	err = t.Execute(f, constClasses)
	if err != nil {
		log.Print("execute: ", err)
		return
	}

	f.Close()
}

func generateOutputDirectory() {
	fmt.Println("Creating output directory")
	_ = os.Mkdir(outputDirectoryPath, 0777)
}

func validateArgs() {
	if len(os.Args) != 3 {
		log.Fatal(`Generate a class of usefull consts using a Unity Project. Generates consts for Layers, Tags, Scenes, Prefabs, GameObjects and Animator Parameters.

Usage :            
	 HarmonyCodeGenerator.py inputDir outputDir
Where :
	inputDir                Path to the Unity project directory (not the Asset folder).
	outputDir               Path to the output directory for generated code (in the Asset folder).`)
	}
}

func generateHashFromString(name string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(name))
	return h.Sum32()
}

func createConstMember(name string, path string) ConstMember {
	isAlphaNumerical, _ := regexp.MatchString("^[a-zA-Z0-9]+$", name)
	isValid := isAlphaNumerical && name != "GameObject" && name != "Scene" && name != "Prefab" && name != "Layer" && name != "Tag" && name != "AnimatorParameter"
	return ConstMember{Name: name, Path: path, Id: generateHashFromString(name), IsValid: isValid}
}

func removeNilFrom(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
