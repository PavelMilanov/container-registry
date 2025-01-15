import { createSignal, onMount, lazy } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"

const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Image() {
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
    const submitDelete = async () => {
        setModalDeleteOpen(false)
        // const headers = {
        //     'Authorization': `Bearer ${TOKEN}`
        // }
        const response = await axios.delete(API_URL + `registry/${params.name}/${image()}`, { params: { "tag": tag() } })
        setTagList(tagList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    async function getImages() {
        const response = await axios.get(API_URL + `registry/${params.name}/${params.image}`)
        setTagList(response.data.data)// в ответе приходит массив "data"
    }

    onMount(async () => {
        await getImages()
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'/'} <A href={"/registry/" + params.name}>{params.name}</A> {'/'} {params.image} </h2>
            <div class="card">
                <Delete isOpen={isModalDeleteOpen()} message={"Образ Docker будет удален!"} onClose={closeDeleteModal} onSubmit={submitDelete} />
                <table>
                    <thead>
                        <tr>
                            <th>Образ</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={tagList()} >{(tag, i) =>
                            <tr>
                                <td>
                                    {tag.Name}:{tag.Tag}
                                </td>
                                <td>{tag.Size}</td>
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