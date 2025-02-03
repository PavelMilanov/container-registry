import { createSignal, onMount, lazy } from "solid-js"
import { A, useParams, useNavigate } from "@solidjs/router"
import axios from "axios"

import NavBar from "./NavBar"
const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Repo() {
    const navigate = useNavigate()
    const [imageList, setImageList] = createSignal([])
    const params = useParams()
    const [repo, setRepo] = createSignal('')
    const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false)
    const openDeleteModal = (item) => {
        setModalDeleteOpen(true)
        setRepo(item)
    }
    const closeDeleteModal = () => setModalDeleteOpen(false)
    let copyText = `${API_URL}/${params.name}`.split("//")[1]

    const submitDelete = async () => {
        setModalDeleteOpen(false)
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        const response = await axios.delete(API_URL + `/api/registry/${params.name}/${repo()}`, {headers: headers})
        setImageList(imageList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    async function getRepo() {
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.get(
                API_URL + `/api/registry/${params.name}`,
                { headers: headers }
            )
            setImageList(response.data.data)  // в ответе приходит массив "data"
        } catch (error) {
            console.log(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", { replace: true })
            }
        }
    }

    onMount(async () => {
        await getRepo()
    })
    return (
        <>
        <NavBar />
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'/'} {params.name}</h2>
            <div class="copy-container">
                <input
                    type="text"
                    value={copyText}
                    readonly
                />
                {/* <button class="btn btn-secondary" onclick="copyText()">Копировать</button>
                <span class="checkmark">✓</span> */}
            </div>
            <div class="card">
                <Delete isOpen={isModalDeleteOpen()} message={"Образы Docker репозитория будут удалены!"} onClose={closeDeleteModal} onSubmit={submitDelete} />
                <table>
                    <thead>
                        <tr>
                            <th>Репозиторий</th>
                            {/* <th>Размер</th> */}
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={imageList()} >{(image, i) =>
                            <tr>
                                <td>
                                    <A href={image.Name}>{image.Name}</A>
                                </td>
                                {/* <td>{repo.Size}</td> */}
                                <td>{image.CreatedAt}</td>
                                <td>
                                    <button class="btn btn-secondary" onClick={() => openDeleteModal(image.Name)}>Удалить репозиторий</button>
                                </td>
                            </tr>
                        }</For>
                    </tbody>
                </table>
            </div>
            </div>
        </>
    )
}

export default Repo