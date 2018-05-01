package main

import (
	"strconv"
	"strings"
)

func nvl(str string) string {
	if str == "" {
		return "-"
	}
	return str
}

func existInt(list []int, i int) bool {
	for _, v := range list {
		if v == i {
			return true
		}
	}
	return false
}

func existStr(list []string, str string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func concatInt(list []int, delimiter string) string {
	return concatIntWithBracket(list, delimiter, "")
}

func concatIntWithBracket(list []int, delimiter, bracket string) string {
	return concatIntWith2Brackets(list, delimiter, bracket, bracket)
}

func concatIntWith2Brackets(list []int, delimiter, bracketFront, bracketBack string) string {
	var str string
	for i := 0; i < len(list); i++ {
		str += bracketFront + strconv.Itoa(list[i]) + bracketBack + delimiter
	}
	return strings.TrimRight(str, delimiter)
}

func concatStr(list []string, delimiter string) string {
	return concatStrWithBracket(list, delimiter, "")
}

func concatStrWithBracket(list []string, delimiter, bracket string) string {
	return concatStrWith2Brackets(list, delimiter, bracket, bracket)
}

func concatStrWith2Brackets(list []string, delimiter, bracketFront, bracketBack string) string {
	var str string
	for i := 0; i < len(list); i++ {
		str += bracketFront + list[i] + bracketBack + delimiter
	}
	return strings.TrimRight(str, delimiter)
}

func space(cnt int) string {
	str := ""
	for i := 0; i < cnt; i++ {
		str += " "
	}
	return str
}
