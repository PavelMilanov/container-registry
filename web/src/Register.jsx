import { createSignal, Show } from "solid-js";
import axios from "axios";

export default function Register() {
  const [errorMessage, setErrorMessage] = createSignal("");
  const [username, setUsername] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [confirmPassword, setConfirmPassword] = createSignal("");

  async function register() {
    let data = {
      username: username(),
      password: password(),
      confirmPassword: confirmPassword(),
    };
    try {
      const response = await axios.post(
        API_URL + "/registration",
        JSON.stringify(data),
      );
      toLogin();
    } catch (error) {
      const msg = error.response.data.error;
      setErrorMessage(msg);
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === "Enter") {
      register();
    }
  };

  return (
    <div class="flex justify-center items-center h-screen">
      <div class="relative bg-white rounded-lg shadow-lg w-1/6">
        <div class="flex bg-main items-center justify-between p-4 md:p-5 border-b rounded-t border-gray-200">
          <h3 class="text-white text-xl font-semibold">
            Регистрация в системе
          </h3>
        </div>

        <div class="p-4 md:p-5">
          <div class="space-y-4">
            <div>
              <label
                for="username"
                class="block mb-2 text-sm font-medium text-gray-900"
              >
                Имя пользователя
              </label>
              <input
                type="text"
                name="username"
                id="username"
                onInput={(e) => setUsername(e.target.value)}
                onKeyDown={handleKeyDown}
                class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                placeholder=""
                required
              />
            </div>
            <div>
              <label
                for="password"
                class="block mb-2 text-sm font-medium text-gray-900"
              >
                Пароль
              </label>
              <input
                type="password"
                name="password"
                id="password"
                placeholder="••••••••"
                onInput={(e) => setPassword(e.target.value)}
                onKeyDown={handleKeyDown}
                class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                required
              />
            </div>
            <div>
              <label
                for="confirmPassword"
                class="block mb-2 text-sm font-medium text-gray-900"
              >
                Пароль
              </label>
              <input
                type="password"
                name="confirmPassword"
                id="confirmPassword"
                placeholder="••••••••"
                onInput={(e) => setConfirmPassword(e.target.value)}
                onKeyDown={handleKeyDown}
                class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                required
              />
            </div>
            <button
              type="submit"
              onClick={register}
              class="w-full text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
            >
              Зарегистрироваться
            </button>
            <div class="text-sm font-medium text-gray-500">
              вернуться
              <a href="/login" class="text-blue-700 hover:underline">
                назад
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
