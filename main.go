package main
var cache = make(map[string]string) // Глобальная переменная для кэша
func main() {
    // Инициализация кэша (пример тестовой инициализации)
    cache["order_12345"] = "processed"
    cache["order_67890"] = "pending"

    // Печать содержимого кэша
    printCache()

    // Проверка наличия конкретного ключа
    checkCache("order_12345")
    checkCache("order_99999")
}
