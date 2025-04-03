import { createSignal, Switch, Match } from "solid-js";
import { useNavigate } from "@solidjs/router";
import axios from "axios";

export default function Login() {
  const navigate = useNavigate();
  const [errorMessage, setErrorMessage] = createSignal("");
  const [username, setUsername] = createSignal("");
  const [password, setPassword] = createSignal("");

  async function login() {
    let data = {
      username: username(),
      password: password(),
    };
    try {
      const response = await axios.post(
        API_URL + "/login",
        JSON.stringify(data),
      );
      localStorage.setItem("token", response.data.token);
      navigate("/registry", { replace: true });
    } catch (error) {
      const msg = error.response.data.error;
      setErrorMessage(msg);
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === "Enter") {
      login();
    }
  };

  return (
    <div class="flex justify-center items-center h-screen">
      <div class="relative bg-white rounded-lg shadow-lg w-1/5">
        <div class="flex bg-main items-center justify-between p-4 md:p-5 border-b rounded-t border-gray-200">
          <h3 class="text-white text-xl font-semibold">Вход в систему</h3>
        </div>
        <div class="p-4 md:p-5">
          <div class="space-y-4">
            <Switch>
              <Match when={!errorMessage()}>
                <div>
                  <label
                    for="username"
                    class="block mb-2 text-sm font-medium text-gray-900"
                  >
                    Логин
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
              </Match>
              <Match when={errorMessage()}>
                <div>
                  <label
                    for="username"
                    class="block mb-2 text-sm font-medium text-gray-900"
                  >
                    Логин
                  </label>
                  <input
                    type="text"
                    name="username"
                    id="username"
                    onInput={(e) => setUsername(e.target.value)}
                    onKeyDown={handleKeyDown}
                    class="bg-red-50 border border-red-500 text-red-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                    placeholder=""
                    required
                  />
                </div>
              </Match>
            </Switch>
            <Switch>
              <Match when={!errorMessage()}>
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
              </Match>
              <Match when={errorMessage()}>
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
                    class="bg-red-50 border border-red-500 text-red-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                    required
                  />
                </div>
                <p class="mt-2 text-sm text-red-600">
                  <span class="font-medium">{errorMessage()}</span>
                </p>
              </Match>
            </Switch>
            <button
              type="submit"
              onClick={login}
              class="w-full text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
            >
              Войти
            </button>
            <div class="text-sm font-medium text-gray-500">
              Не зарегистрированы?
              <a href="/register" class="text-blue-700 hover:underline">
                Зарегистрироваться
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
