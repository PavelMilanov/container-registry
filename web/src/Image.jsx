import { createSignal, onMount, lazy } from "solid-js"
import { A, useParams, useNavigate } from "@solidjs/router"
import axios from "axios"

import NavBar from "./NavBar"
const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Image() {
    const navigate = useNavigate()
    const [tagList, setTagList] = createSignal([])
    const params = useParams()
    const [tag, setTag] = createSignal('')
    const [image, setImage] = createSignal('')
    const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false)
    const openDeleteModal = (img, tag) => {
        setModalDeleteOpen(true)
        setImage(img)
        setTag(tag)
    }
    const closeDeleteModal = () => setModalDeleteOpen(false)

    let copyText = `${API_URL}/${params.name}/${params.image}`.split("//")[1]

    const submitDelete = async () => {
        setModalDeleteOpen(false)
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.delete(
                API_URL + `/api/registry/${params.name}/${image()}`,
                { headers: headers, params: { "tag": tag() } }
            )
            await getImages()
        } catch (error) {
            console.error(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", { replace: true })
            }
        }
    }

    async function getImages() {
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.get(
                API_URL + `/api/registry/${params.name}/${params.image}`,
                { headers: headers }
            )
            setTagList(response.data.data)// в ответе приходит массив "data"
        } catch (error) {
            console.log(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", { replace: true })
            }
        }
    }

    onMount(async () => {
        await getImages()
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a>{"/"}<A href={"/registry/" + params.name}>{params.name}</A> {"/"} {params.image} </h2>
            <div class="card">
                <Delete isOpen={isModalDeleteOpen()} message={"Образ Docker будет удален!"} onClose={closeDeleteModal} onSubmit={submitDelete} />
                <table>
                    <thead>
                        <tr>
                            <th>Образ</th>
                            <th>Хеш</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={tagList()} >{(tag, i) =>
                            <tr>
                                <td>
                                    {copyText()}:{tag.Tag}
                                </td>
                                    <td>{tag.Hash.slice(0, 15)}...</td>
                                <td>{tag.SizeAlias}</td>
                                <td>{tag.CreatedAt}</td>
                                <td>
                                    <button class="btn btn-secondary" onClick={() => openDeleteModal(tag.Name, tag.Tag)}>Удалить образ</button>
                                </td>
                            </tr>
                        }</For>
                    </tbody>
                </table>
            </div>
        </div>
    )
}

export default Image