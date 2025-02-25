import { createSignal, onMount } from "solid-js"
import { query, useNavigate } from "@solidjs/router"
import toast from 'solid-toast'
import axios from "axios"

const API_URL = window.API_URL

function Settings() {
    const navigate = useNavigate()
    const [version, setVersion] = createSignal("")

    async function garbageCollection() {
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.post(
                API_URL + "/api/settings?garbage=true",
                {},
                { headers: headers}
            )
            if (response.status === 202) {
                toast(response.data.data, {
                    style: {
                        "background-color": "#1e3c72",
                        "color": "white"
                    },
                    className: "notification info",
                })
            }
        } catch (error) {
            console.log(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", { replace: true })
            }
        }
    }

    async function getSettings() {
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.get(
                API_URL + "/api/settings",
                { headers: headers }
            )
            setVersion(response.data.version)
        } catch (error) {
            console.log(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", { replace: true })
            }
        }
    }

    onMount(async () => {
            await getSettings()
        })

    return (
        <div class="container">
            <h2>Общие настройки</h2>
            <div class="card">
                {/* <div class="form-group">
                    <label for="BackupCount">Количество бэкапов на диске:</label>
                    <input type="text" id="BackupCount" value="{{.config.BackupCount}}" name="BackupCount" required />
                </div> */}
                <div class="form-group">
                    <p>Удалить неиспользуемые файлы реестра:</p>
                    <button class="btn btn-primary" onClick={garbageCollection}>Garbage Collection</button>
                </div>
                <div class="form-group">
                    <label for="version">Версия сборки:</label>
                    <p name="version">{version()}</p>
                </div>
                {/* <button type="submit" class="btn btn-primary">Сохранить общие настройки</button> */}
            </div>
        </div>
    )
}

export default Settings