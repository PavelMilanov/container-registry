import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import axios from "axios";

function Login() {
  const navigate = useNavigate();
  const [errorMessage, setErrorMessage] = createSignal("");
  const [username, setUsername] = createSignal("");
  const [password, setPassword] = createSignal("");

  function toRegister() {
    navigate("/register");
  }

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
    <div class="login-container" id="app">
      <h1>Вход в систему</h1>
      <label for="username">Имя пользователя:</label>
      <input
        class="login-input"
        id="username"
        type="text"
        required
        onInput={(e) => setUsername(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      <label for="password">Пароль:</label>
      <input
        class="login-input"
        id="password"
        type="password"
        required
        onInput={(e) => setPassword(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      <Show when={errorMessage()}>
        <p class="error-message">{errorMessage()}</p>
      </Show>
      <button class="btn login-button" onClick={login}>
        Войти
      </button>
      <div class="divider">
        <span>или</span>
      </div>
      <button class="btn register-button" onClick={toRegister}>
        Зарегистрироваться
      </button>
    </div>
  );
}

export default Login;
