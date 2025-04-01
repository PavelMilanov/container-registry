import { createSignal } from "solid-js";

export default function Clipboard() {
  let inputRef;
  const [copied, setCopied] = createSignal(false);

  const copyToClipboard = async () => {
    const text = inputRef.value; // Получаем значение поля ввода
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      // Скрываем сообщение через 2 секунды
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Ошибка копирования: ", error);
    }
  };

  return (
    <div class="w-full max-w-[24rem] ml-4">
      <div class="relative">
        <label for="clipboard-input" class="sr-only">
          Label
        </label>
        <input
          id="clipboard-input"
          type="text"
          class="bg-blue-50 text-xs rounded-lg focus:ring-blue-800 focus:border-blue-800 block w-full px-2.5 py-4"
          value={API_URL.split("//")[1]}
          disabled
          readonly
          ref={inputRef}
        />
        <button
          onClick={copyToClipboard}
          class="absolute end-2.5 top-1/2 -translate-y-1/2 text-xs hover:bg-gray-100 rounded-lg py-2 px-2.5 inline-flex items-center justify-center bg-white border border-gray-200 h-8"
        >
          {!copied() ? (
            <span id="default-message" class="inline-flex items-center">
              <svg
                class="w-3 h-3 me-1.5"
                aria-hidden="true"
                xmlns="http://www.w3.org/2000/svg"
                fill="currentColor"
                viewBox="0 0 18 20"
              >
                <path d="M16 1h-3.278A1.992 1.992 0 0 0 11 0H7a1.993 1.993 0 0 0-1.722 1H2a2 2 0 0 0-2 2v15a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2Zm-3 14H5a1 1 0 0 1 0-2h8a1 1 0 0 1 0 2Zm0-4H5a1 1 0 0 1 0-2h8a1 1 0 1 1 0 2Zm0-5H5a1 1 0 0 1 0-2h2V2h4v2h2a1 1 0 1 1 0 2Z" />
              </svg>
              <span class="text-xs font-semibold">Скопировать</span>
            </span>
          ) : (
            <span id="success-message" class="inline-flex items-center">
              <svg
                class="w-3 h-3 text-base me-1.5"
                aria-hidden="true"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 16 12"
              >
                <path
                  stroke="currentColor"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M1 5.917 5.724 10.5 15 1.5"
                />
              </svg>
              <span class="text-xs font-semibold">Скопировано!</span>
            </span>
          )}
        </button>
      </div>
    </div>
  );
}
