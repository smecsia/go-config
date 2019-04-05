package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
	. "github.com/smecsia/go-utils/pkg/util"
)

const (
	envTag     = "env"
	defaultTag = "default"
	yamlTag    = "yaml"
)

type Config interface {
	SetConfigFilePath(path string)
	GetConfigFilePath() string
	Init() error
}


// ReadConfig reads config from console based on provided one
func ReadConfig(defaultConfig Config, reader ConsoleReader) Config {
	fields := reflect.ValueOf(defaultConfig).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		if len(field.String()) == 0 && field.CanSet() {
			fieldName := fields.Type().Field(i).Name
			fmt.Printf("Enter %s [%s]: ", fieldName, field)
			var text string
			if fieldName == "Password" {
				text, _ = reader.ReadPassword()
			} else {
				text, _ = reader.ReadLine()
			}
			if len(text) > 0 && text != "\n" {
				setFieldValue(field, text)
			}
		}
	}
	return defaultConfig
}

// AddDefaults sets default values into fields not defined in raw config
func AddDefaults(rawConfig map[string]interface{}, newConfig Config) Config {
	fieldsOld := reflect.ValueOf(newConfig).Elem()
	typeOfConfig := reflect.TypeOf(newConfig).Elem()
	for i := 0; i < fieldsOld.NumField(); i++ {
		field := fieldsOld.Field(i)
		fieldType := typeOfConfig.Field(i)
		yamlTagName := getYamlFieldName(fieldType)
		defaultValue := fieldType.Tag.Get(defaultTag)
		if rawConfig[yamlTagName] == nil || yamlTagName == "-" {
			setFieldValue(field, defaultValue)
		}
	}
	return newConfig
}

// AddEnv sets values from env (if any)
func AddEnv(newConfig Config) Config {
	fieldsOld := reflect.ValueOf(newConfig).Elem()
	typeOfConfig := reflect.TypeOf(newConfig).Elem()
	for i := 0; i < fieldsOld.NumField(); i++ {
		field := fieldsOld.Field(i)
		fieldType := typeOfConfig.Field(i)
		envVarName := fieldType.Tag.Get(envTag)
		envValue := os.Getenv(envVarName)
		if envValue != "" {
			setFieldValue(field, envValue)
		}
	}
	return newConfig
}

func getYamlFieldName(fieldType reflect.StructField) string {
	return strings.Split(fieldType.Tag.Get(yamlTag), ",")[0]
}

// DefaultConfig Returns default version of Config file
func DefaultConfig(cfgObj Config) Config {
	return AddDefaults(map[string]interface{}{}, cfgObj)
}

func setFieldValue(field reflect.Value, valueString string) {
	if !field.CanSet() {
		return
	}
	if isStringType(field) {
		field.Set(reflect.ValueOf(valueString))
	} else if isIntType(field) {
		if intVal, e := strconv.Atoi(valueString); e != nil {
			fmt.Println(fmt.Sprintf("WARN: '%s' is not integer", valueString))
		} else {
			field.Set(reflect.ValueOf(int64(intVal)))
		}
	} else if isBoolType(field) {
		field.Set(reflect.ValueOf(valueString == "true"))
	}

}

// ReadConfigFile Reads config file from yaml file
func ReadConfigFile(filePath string, readConfig Config) (Config, map[string]interface{}, error) {
	rawConfig := make(map[string]interface{})
	if fileBytes, err := ioutil.ReadFile(filePath); err == nil {
		err = yaml.Unmarshal(fileBytes, readConfig)
		if err != nil {
			return readConfig, rawConfig, err
		}
		err = yaml.Unmarshal(fileBytes, &rawConfig)
		if err != nil {
			return readConfig, rawConfig, err
		}
	}
	readConfig.SetConfigFilePath(filePath)
	return readConfig, rawConfig, nil
}

// Writes config file to yaml file
func WriteConfigFile(filePath string, cfg Config) error {
	if fileBytes, err := yaml.Marshal(cfg); err != nil {
		return err
	} else if err := ioutil.WriteFile(filePath, fileBytes, os.ModePerm); err != nil {
		return err
	}
	return nil
}

// Reads config file from yaml safely and adds defaults from env or default tags
func Init(filePath string, cfgObj Config, reader ConsoleReader) Config {
	config, rawConfig, err := ReadConfigFile(filePath, cfgObj)
	if err != nil {
		panic(err)
	}
	cfg := ReadConfig(AddEnv(AddDefaults(rawConfig, config)), reader)
	if err := cfg.Init(); err != nil {
		panic(err)
	}
	return cfg
}

func isStringType(field reflect.Value) bool {
	return field.Type() == reflect.TypeOf("")
}

func isIntType(field reflect.Value) bool {
	return field.Type() == reflect.TypeOf(int64(0))
}

func isBoolType(field reflect.Value) bool {
	return field.Type() == reflect.TypeOf(false)
}
