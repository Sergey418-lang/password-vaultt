package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Entry struct {
	Service  string
	Login    string
	Password string
}

var (
	vault    []Entry
	key      = getKey()
	fileName = "vault.json"
)

func getKey() string {
	if k := os.Getenv("VAULT_KEY"); k != "" {
		return k
	}
	return "default-key"
}

func crypt(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		result[i] = s[i] ^ key[i%len(key)]
	}
	return string(result)
}

func save() {
	data, _ := json.MarshalIndent(vault, "", "  ")
	os.WriteFile(fileName, data, 0600)
}

func load() {
	data, _ := os.ReadFile(fileName)
	json.Unmarshal(data, &vault)
}

func checkMaster() bool {
	hashFile := "master.hash"
	savedHash, err := os.ReadFile(hashFile)

	if os.IsNotExist(err) {
		fmt.Print("Придумайте мастер-пароль: ")
		pass := readInput()
		fmt.Print("Подтвердите: ")
		if pass != readInput() || len(pass) < 4 {
			fmt.Println("Ошибка!")
			return false
		}
		hash := sha256.Sum256([]byte(pass))
		os.WriteFile(hashFile, []byte(hex.EncodeToString(hash[:])), 0600)
		fmt.Println("Готово!")
		return true
	}

	fmt.Print("Мастер-пароль: ")
	pass := readInput()
	hash := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(hash[:]) == strings.TrimSpace(string(savedHash))
}

func readInput() string {
	text, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(text)
}

func addEntry() {
	fmt.Print("Сервис: ")
	service := readInput()
	fmt.Print("Логин: ")
	login := readInput()
	fmt.Print("Пароль: ")
	password := readInput()

	if service == "" || login == "" || password == "" {
		fmt.Println("Все поля обязательны!")
		return
	}

	vault = append(vault, Entry{
		Service:  service,
		Login:    login,
		Password: crypt(password),
	})
	save()
	fmt.Println("Добавлено!")
}

func listEntries() {
	if len(vault) == 0 {
		fmt.Println("Пусто")
		return
	}
	for i, e := range vault {
		fmt.Printf("[%d] %s | %s | %s\n", i, e.Service, e.Login, crypt(e.Password))
	}
}

func searchEntries() {
	fmt.Print("Что ищем: ")
	term := strings.ToLower(readInput())
	found := false

	for i, e := range vault {
		if strings.Contains(strings.ToLower(e.Service), term) {
			fmt.Printf("[%d] %s | %s | %s\n", i, e.Service, e.Login, crypt(e.Password))
			found = true
		}
	}
	if !found {
		fmt.Println("Ничего не найдено")
	}
}

func deleteEntry() {
	if len(vault) == 0 {
		fmt.Println("Нечего удалять")
		return
	}

	listEntries()
	fmt.Print("Номер для удаления: ")
	var idx int
	fmt.Scanln(&idx)

	if idx < 0 || idx >= len(vault) {
		fmt.Println("Неверный номер")
		return
	}

	fmt.Printf("Удалить %s? (y/n): ", vault[idx].Service)
	if strings.ToLower(readInput()) == "y" {
		vault = append(vault[:idx], vault[idx+1:]...)
		save()
		fmt.Println("Удалено!")
	}
}

func generatePassword() {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	length := 12
	password := make([]byte, length)

	for i := 0; i < length; i++ {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		password[i] = chars[int(randomByte[0])%len(chars)]
	}

	fmt.Printf("Пароль: %s\n", string(password))
}

func exportCSV() {
	file, _ := os.Create("export.csv")
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	w.Write([]string{"Service", "Login", "Password"})
	for _, e := range vault {
		w.Write([]string{e.Service, e.Login, crypt(e.Password)})
	}
	fmt.Println("Экспортировано в export.csv")
}

func importCSV() {
	file, err := os.Open("export.csv")
	if err != nil {
		fmt.Println("Файл export.csv не найден")
		return
	}
	defer file.Close()

	records, _ := csv.NewReader(file).ReadAll()
	count := 0

	for i := 1; i < len(records); i++ {
		r := records[i]
		if len(r) >= 3 {
			vault = append(vault, Entry{
				Service:  r[0],
				Login:    r[1],
				Password: crypt(r[2]),
			})
			count++
		}
	}

	save()
	fmt.Printf("Импортировано %d записей\n", count)
}

func showMenu() {
	options := []string{
		"1. Добавить",
		"2. Показать все",
		"3. Найти",
		"4. Удалить",
		"5. Сгенерировать пароль",
		"6. Экспорт в CSV",
		"7. Импорт из CSV",
		"8. Выход",
	}

	fmt.Println("\n" + strings.Repeat("-", 30))
	for _, opt := range options {
		fmt.Println(opt)
	}
	fmt.Print("Выберите: ")
}

func main() {
	if !checkMaster() {
		return
	}

	load()

	for {
		showMenu()
		choice := readInput()

		switch choice {
		case "1":
			addEntry()
		case "2":
			listEntries()
		case "3":
			searchEntries()
		case "4":
			deleteEntry()
		case "5":
			generatePassword()
		case "6":
			exportCSV()
		case "7":
			importCSV()
		case "8":
			fmt.Println("До свидания!")
			return
		default:
			fmt.Println("Выберите от 1 до 8")
		}
	}
}