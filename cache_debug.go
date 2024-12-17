package main

import "log"

// Печать всего содержимого кэша
func printCache() {
    log.Println("Текущее содержимое кэша:")
    for key, value := range cache {
        log.Printf("Key: %s, Value: %s", key, value)
    }
}

// Проверка наличия конкретного ключа в кэше
func checkCache(orderUID string) {
    if value, found := cache[orderUID]; found {
        log.Printf("Key '%s' найден в кэше со значением: %s", orderUID, value)
    } else {
        log.Printf("Key '%s' не найден в кэше", orderUID)
    }
}
