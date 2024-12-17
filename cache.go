package main

import "log"

// Глобальная переменная для кэша
// var cache = make(map[string]string) // Ключ: order_uid, Значение: статус заказа

// Функция обновления кэша
func updateCache(orderUID string) {
    cache[orderUID] = "default" // Заполняем кэш значением по умолчанию
    log.Printf("Кэш обновлён: %v", cache)
}

func getFromCache(orderUID string) bool {
    _, found := cache[orderUID]
    return found
}


// Функция инициализации кэша из базы данных
func initializeCache(db DBConnection) error {
    rows, err := db.Query("SELECT order_uid FROM orders")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var orderUID string
        if err := rows.Scan(&orderUID); err != nil {
            return err
        }
        cache[orderUID] = "default" // Записываем статическое значение в кэш
    }

    log.Println("Кэш успешно инициализирован из базы данных")
    return nil
}

