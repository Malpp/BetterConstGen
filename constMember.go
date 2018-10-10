package main

import (
	"crypto/md5"
	"encoding/hex"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

type ConstMember struct {
	Id      uint64
	Name    string
	Value   string
	Path    string
	IsValid bool
}

func generateHashFromString(name string) uint64 {
	data := []byte(name)
	sum := md5.Sum(data)
	hexDigest := hex.EncodeToString(sum[:])
	s := string(hexDigest)
	i := new(big.Int)
	i.SetString(s, 16)
	modNum := new(big.Int)
	modNum.SetInt64(100000000)
	i.Mod(i, modNum)
	return i.Uint64()
}

func createConstMember(name string, path string) ConstMember {
	valS := name
	name = strings.Replace(name, "_", "", -1)
	isAlphaNumerical, _ := regexp.MatchString("^[a-zA-Z0-9]+$", name)
	isValid := isAlphaNumerical && name != "GameObject" && name != "Scene" && name != "Prefab" && name != "Layer" && name != "Tag" && name != "AnimatorParameter" && !unicode.IsDigit(rune(name[0]))
	return ConstMember{Name: name, Path: path, Id: generateHashFromString(name), IsValid: isValid, Value: valS}
}
