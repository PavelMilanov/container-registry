import { Show } from "solid-js";
import { alertMessage, alertType, setAlertMessage } from "./alertService";

export default function Alert() {
  return (
    <Show when={alertMessage()}>
      <div
        id="alert-border-1"
        class={`fixed top-50 right-4 z-50 flex items-center p-4 rounded-lg ${
          alertType() === "error" ? "bg-red-100" : "bg-blue-100"
        }`}
        role="alert"
      >
        <svg
          class={`flex-shrink-0 w-5 h-5 text-${
            alertType() === "error" ? "red" : "blue"
          }-700`}
          aria-hidden="true"
          xmlns="http://www.w3.org/2000/svg"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5ZM9.5 4a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3ZM12 15H8a1 1 0 0 1 0-2h1v-3H8a1 1 0 0 1 0-2h2a1 1 0 0 1 1 1v4h1a1 1 0 0 1 0 2Z" />
        </svg>
        <div class="ml-3 text-sm font-medium">{alertMessage()}</div>
        <button
          type="button"
          class={`ml-auto -mx-1.5 -my-1.5 bg-${
            alertType() === "error" ? "red" : "blue"
          }-100 text-${
            alertType() === "error" ? "red" : "blue"
          }-500 rounded-lg focus:ring-2 focus:outline-none p-1.5 hover:bg-${
            alertType() === "error" ? "red" : "blue"
          }-200`}
          onClick={() => setAlertMessage("")}
        >
          <span class="sr-only">Dismiss</span>
          <svg
            class="w-3 h-3"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 14 14"
          >
            <path
              stroke="currentColor"
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"
            />
          </svg>
        </button>
      </div>
    </Show>
  );
}
