import { createSignal, Show } from 'solid-js'

function Login() {
    const [loggedIn, setLoggedIn] = createSignal(false)
    const toggle = () => setLoggedIn(!loggedIn())

    return (
        <div class="login-container" id="app">
            <Show
                when={loggedIn()}
                fallback={() =>
                <>
                <h1>Вход в систему</h1>
                <label for="username">Имя пользователя:</label>
                <input type="text" id="username" name="Username" required />
                <label for="password">Пароль:</label>
                <input type="password" id="password" name="Password" required />
                <button class="btn login-button">
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
                    <input type="text" id="username" name="Username" required />
                    <label for="password">Пароль:</label>
                    <input type="password" id="password" name="Password" required />
                    <label for="confirmPassword">Подтвердите пароль:</label>
                    <input type="password" id="confirmPassword" name="Password" required />
                    <button class="btn login-button" onClick={toggle}>
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
