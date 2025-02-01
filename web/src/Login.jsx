import { createSignal, Show } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import axios from 'axios'

function Login() {
    const navigate = useNavigate()
    const [loggedIn, setLoggedIn] = createSignal(false)
    const toggle = () => setLoggedIn(!loggedIn())

    const [username, setUsername] = createSignal('')
    const [password, setPassword] = createSignal('')
    const [confirmPassword, setConfirmPassword] = createSignal('')

    async function register() {
        let data = {
            username: username(),
            password: password(),
            confirmPassword: confirmPassword(),
        }
        try {
            const response = await axios.post(API_URL + "/registration", JSON.stringify(data))
            console.log(response.data)
        } catch (error) { 
            console.log("ошибка", error)
        }
    }

    async function login() { 
        let data = {
            username: username(),
            password: password(),
        }
        try {
            const response = await axios.post(API_URL + "/login", JSON.stringify(data))
            localStorage.setItem("token", response.data.token)
            navigate("/registry", { replace: true })
        } catch (error) {
            console.log("ошибка", error)
        }
    }

    return (
        <div class="login-container" id="app">
            <Show
                when={loggedIn()}
                fallback={() =>
                <>
                <h1>Вход в систему</h1>
                <label for="username">Имя пользователя:</label>
                <input type="text" id="username" required onInput={(e) => (setUsername(e.target.value))} />
                <label for="password">Пароль:</label>
                <input type="password" id="password" required onInput={(e) => (setPassword(e.target.value))} />
                <button class="btn login-button" onClick={login}>
                    Войти
                </button>
                <div class="divider">
                    <span>или</span>
                </div>
                <button class="btn register-button" onClick={toggle}>
                    Зарегистрироваться
                </button>
                </>
                }
            >
            <>
                <h1>Регистрация в системе</h1>
                <label for="username">Имя пользователя:</label>
                <input type="text" id="username" required onInput={(e) => (setUsername(e.target.value))} />
                <label for="password">Пароль:</label>
                <input type="password" id="password" required onInput={(e) => (setPassword(e.target.value))}/>
                <label for="confirmPassword">Подтвердите пароль:</label>
                <input type="password" id="confirmPassword" required onInput={(e) => (setConfirmPassword(e.target.value))} />
                <button class="btn login-button" onClick={register}>
                    Зарегистрироваться
                </button>
                <div class="divider">
                    <span>или</span>
                </div>
                <button class="btn register-button" onClick={toggle}>
                    Вернуться ко входу
                </button>
            </>
            </Show>
            {/* <p class="error-message">err</p> */}
        </div>
    )
}

export default Login
