import { createSignal } from "solid-js";

// Глобальные сигналы для состояния алерта
export const [alertMessage, setAlertMessage] = createSignal("");
export const [alertType, setAlertType] = createSignal("info"); // "info", "error"

// Глобальная функция для вызова алерта
export function showAlert(message, type = "info") {
  setAlertMessage(message);
  setAlertType(type);
  // Автоматически скрыть через 2 секунд (опционально)
  setTimeout(() => {
    setAlertMessage("");
  }, 3000);
}
