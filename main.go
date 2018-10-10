package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var projectDirectoryPath string
var outputDirectoryPath string

type ConstClass struct {
	Name    string
	Members []ConstMember
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
var animItems []FolderItem

func main() {
	validateArgs()
	generateOutputDirectory()

	fmt.Println("Reading files")
	addConstMembersFromTagAndLayer()

	prepareSceneAndPrefabItems()

	addSceneToConstMembers()
	addPrefabToConstMembers()
	addAnimsToConstMembers()
	addAnimationParametersToConstMembers()
	addGameObjectsToConstMembers()

	generateTemplate()
}

func addGameObjectsToConstMembers() {
	fmt.Println("	GameObjects...")

	var constMembers = make(map[string]string)
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
			constMembers[gameObjectYaml.GameObject.MName] = controllerItem.Path
		}
	}
	constMembers["None"] = ""

	var constMembersData []ConstMember

	for key, value := range constMembers {
		constMembersData = append(constMembersData, createConstMember(key, value))
	}
	constClasses = append(constClasses, ConstClass{Name: "GameObject", Members: constMembersData})
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
	constClasses = append(constClasses, ConstClass{Name: "AnimatorParameter", Members: constMembers})
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

			if strings.HasSuffix(info.Name(), ".anim") {
				animItems = append(animItems, FolderItem{Name: strings.Split(info.Name(), ".")[0], Path: path})
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

func addAnimsToConstMembers() {
	fmt.Println("	Animations...")
	constClasses = append(constClasses, ConstClass{Name: "Animations", Members: folderItemsToConstMembers(animItems)})
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
	addConstClassFromMembers("Tag", tags, tagManagerPath)
}

func addLayerConstMembers(layers []string, tagManagerPath string) {
	layers = append(layers, "None")
	addConstClassFromMembers("Layer", layers, tagManagerPath)
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

	t, err := template.New("cSharp").Parse(cSharpTemplate)
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

	projectDirectoryPath = os.Args[1]
	outputDirectoryPath = os.Args[2]
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

var cSharpTemplate = `// ----- AUTO GENERATED CODE - ANY MODIFICATION WILL BE OVERRIDEN ----- //
// ----- GENERATED ON ${timeStamp} ----- //
using System;

namespace Harmony
{
    public static class R
    {
        public static class E
        {
        {{range $constClass := .}}
            public enum {{$constClass.Name}}
            {
            {{range $constClass.Members}}
                {{if .IsValid}}
                {{.Name}} = {{.Id}}, //In "{{.Path}}".
                {{else}}
                //{{$constClass.Name}} "{{.Name}}" has invalid name. Non-alphanumerical characters are prohibited. In "{{.Path}}".
                {{end}}
            {{end}}
            }
        {{end}}
        }
        public static class S
        {
        {{range $constClass := .}}
            public static class {{$constClass.Name}}
            {
            {{range $constClass.Members}}
                {{if .IsValid}}
                public const string {{.Name}} = "{{.Value}}"; //In "{{.Path}}".
                {{else}}
                //{{$constClass.Name}} "{{.Name}}" has invalid name. Non-alphanumerical characters are prohibited. In "{{.Path}}".
                {{end}}
            {{end}}

                public static string ToString(E.{{$constClass.Name}} value)
                {
                    switch (value)
                    {
                    {{range $constClass.Members}}
                        {{if .IsValid}}
                        case E.{{$constClass.Name}}.{{.Name}}:
                            return {{.Name}};
                        {{end}}
                    {{end}}
                    }
                    return null;
                }

                public static E.{{$constClass.Name}} ToEnum(string value)
                {
                    switch (value)
                    {
                    {{range $constClass.Members}}
                        {{if .IsValid}}
                        case {{.Name}}:
                            return E.{{$constClass.Name}}.{{.Name}};
                        {{end}}
                    {{end}}
                    }
                    throw new ArgumentException("Unable to convert " + value + " to R.E.{{$constClass.Name}}.");
                }
            }
        {{end}}
        }
    }
}`
