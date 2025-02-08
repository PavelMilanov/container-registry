import { createSignal, Show } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import axios from 'axios'

function Register() {
    const navigate = useNavigate()
    const [errorMessage, setErrorMessage] = createSignal('')
    const [username, setUsername] = createSignal('')
    const [password, setPassword] = createSignal('')
    const [confirmPassword, setConfirmPassword] = createSignal('')

    function toLogin() {
        navigate("/login")
    }

    async function register() {
        let data = {
            username: username(),
            password: password(),
            confirmPassword: confirmPassword(),
        }
        try {
            const response = await axios.post(API_URL + "/registration", JSON.stringify(data))
            toLogin()
        } catch (error) {
            const msg = error.response.data.error
            setErrorMessage(msg)
        }
    }

    const handleKeyDown = (e) => {
        if (e.key === "Enter") {
            register()
        }
    }

    return (
        <div class="login-container" id="app">
            <h1>Регистрация в системе</h1>
            <label for="username">Имя пользователя:</label>
            <input
                id="username"
                type="text"
                required
                onInput={(e) => (setUsername(e.target.value))}
                onKeyDown={handleKeyDown}
            />
            <label for="password">Пароль:</label>
            <input
                id="password"
                type="password"
                required
                onInput={(e) => (setPassword(e.target.value))}
                onKeyDown={handleKeyDown}
            />
            <label for="confirmPassword">Подтвердите пароль:</label>
            <input
                id="confirmPassword"
                type="password"
                required
                onInput={(e) => (setConfirmPassword(e.target.value))}
                onKeyDown={handleKeyDown}
            />
            <Show
                when={errorMessage()}
            >
                <p class="error-message">{errorMessage()}</p>
            </Show>
            <button class="btn login-button" onClick={register}>
                Зарегистрироваться
            </button>
            <div class="divider">
                <span>или</span>
            </div>
            <button class="btn register-button" onClick={toLogin}>
                Вернуться ко входу
            </button>
        </div>
    )
}

export default Register
