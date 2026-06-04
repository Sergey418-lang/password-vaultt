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
	Service  string `json:"service"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

var (
	vault    []Entry
	fileName = "vault.json"
	useEncryption = false // false = без шифрования, true = с шифрованием
)

// ========== ШИФРОВАНИЕ (шаг 5) ==========
var key = getKey()

func getKey() string {
	if k := os.Getenv("VAULT_KEY"); k != "" {
		return k
	}
	return "default-key-2024"
}

func crypt(s string) string {
	if !useEncryption {
		return s // если шифрование выключено, возвращаем как есть
	}
	result := make([]byte, len(s))
	for i := range s {
		result[i] = s[i] ^ key[i%len(key)]
	}
	return string(result)
}

// ========== JSON СОХРАНЕНИЕ И ЗАГРУЗКА (шаг 3) ==========
func save() {
	data, _ := json.MarshalIndent(vault, "", "  ")
	os.WriteFile(fileName, data, 0600)
}

func load() {
	data, _ := os.ReadFile(fileName)
	json.Unmarshal(data, &vault)
}

// ========== МАСТЕР-ПАРОЛЬ (шаг 6) ==========
func checkMaster() bool {
	hashFile := "master.hash"
	savedHash, err := os.ReadFile(hashFile)

	if os.IsNotExist(err) {
		fmt.Print("Придумайте мастер-пароль: ")
		pass := read("")
		fmt.Print("Подтвердите: ")
		if pass != read("") || len(pass) < 4 {
			fmt.Println("Ошибка!")
			return false
		}
		hash := sha256.Sum256([]byte(pass))
		os.WriteFile(hashFile, []byte(hex.EncodeToString(hash[:])), 0600)
		fmt.Println("Мастер-пароль установлен!")
		return true
	}

	fmt.Print("Введите мастер-пароль: ")
	pass := read("")
	hash := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(hash[:]) == strings.TrimSpace(string(savedHash))
}

// ========== ВВОД ДАННЫХ ==========
func read(prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

// ========== ДОБАВЛЕНИЕ (шаг 1) ==========
func addEntry() {
	service := read("Сервис: ")
	login := read("Логин: ")
	password := read("Пароль: ")

	if service == "" || login == "" || password == "" {
		fmt.Println("Все поля обязательны!")
		return
	}

	vault = append(vault, Entry{
		Service:  service,
		Login:    login,
		Password: crypt(password), // шифруется только если useEncryption = true
	})
	save()
	fmt.Println("Добавлено!")
}

// ========== ПОИСК (шаг 4) ==========
// Эта функция НЕ ЗАВИСИТ от шифрования
// Она ищет только по названию сервиса
func searchEntries() {
	if len(vault) == 0 {
		fmt.Println("Хранилище пусто")
		return
	}

	term := strings.ToLower(read("Что ищем: "))
	found := false

	fmt.Println("\n=== Результаты поиска ===")
	for i, e := range vault {
		// Поиск ТОЛЬКО по сервису (не по паролю)
		// Сервис НЕ шифруется, поэтому поиск работает всегда
		if strings.Contains(strings.ToLower(e.Service), term) {
			// При показе расшифровываем пароль
			fmt.Printf("[%d] Сервис: %s | Логин: %s | Пароль: %s\n",
				i, e.Service, e.Login, crypt(e.Password))
			found = true
		}
	}

	if !found {
		fmt.Println("Ничего не найдено")
	}
	fmt.Println("=========================")
}

// ========== ПОКАЗАТЬ ВСЕ (шаг 2) ==========
func listEntries() {
	if len(vault) == 0 {
		fmt.Println("Хранилище пусто")
		return
	}

	fmt.Println("\n=== Все записи ===")
	for i, e := range vault {
		fmt.Printf("[%d] Сервис: %s | Логин: %s | Пароль: %s\n",
			i, e.Service, e.Login, crypt(e.Password))
	}
	fmt.Println("==================")
}

// ========== УДАЛЕНИЕ (шаг 4) ==========
// Эта функция НЕ ЗАВИСИТ от шифрования
// Она удаляет по индексу, не трогая пароль
func deleteEntry() {
	if len(vault) == 0 {
		fmt.Println("Нечего удалять")
		return
	}

	listEntries()

	indexStr := read("\nВведите номер для удаления: ")
	var index int
	_, err := fmt.Sscan(indexStr, &index)

	if err != nil || index < 0 || index >= len(vault) {
		fmt.Println("Неверный номер!")
		return
	}

	// Показываем сервис (он не зашифрован)
	fmt.Printf("Удалить '%s'? (да/нет): ", vault[index].Service)
	if strings.ToLower(read("")) != "да" {
		fmt.Println("Отменено")
		return
	}

	// Удаление по индексу - работает одинаково с шифрованием и без
	vault = append(vault[:index], vault[index+1:]...)
	save()
	fmt.Println("Удалено!")
}

// ========== ГЕНЕРАТОР ПАРОЛЕЙ (шаг 7) ==========
func generatePassword() {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	length := 12
	password := make([]byte, length)

	for i := 0; i < length; i++ {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		password[i] = chars[int(randomByte[0])%len(chars)]
	}

	fmt.Printf("Сгенерированный пароль: %s\n", string(password))
}

// ========== ЭКСПОРТ CSV (шаг 8) ==========
func exportCSV() {
	if len(vault) == 0 {
		fmt.Println("Нечего экспортировать")
		return
	}

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

// ========== ИМПОРТ CSV (шаг 8) ==========
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

// ========== МЕНЮ (шаг 2) ==========
func showMenu() {
	fmt.Println("\n=== Менеджер паролей ===")
	fmt.Println("1. Добавить запись")
	fmt.Println("2. Показать все")
	fmt.Println("3. Найти по сервису")
	fmt.Println("4. Удалить запись")
	fmt.Println("5. Сгенерировать пароль")
	fmt.Println("6. Экспорт в CSV")
	fmt.Println("7. Импорт из CSV")
	fmt.Println("8. Выход")
	fmt.Print("\nВыберите: ")
}

// ========== main ==========
func main() {
	if !checkMaster() {
		fmt.Println("Доступ запрещён!")
		return
	}

	load()

	for {
		showMenu()
		choice := read("")

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