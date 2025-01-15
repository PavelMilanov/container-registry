import { createSignal, onMount, lazy } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"

const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Repo() {
    const [imageList, setImageList] = createSignal([])
    const params = useParams()
    const [repo, setRepo] = createSignal('')

    const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false)

    const openDeleteModal = (item) => {
        setModalDeleteOpen(true)
        setRepo(item)
    }
    const closeDeleteModal = () => setModalDeleteOpen(false)
    const submitDelete = async () => {
        setModalDeleteOpen(false)
        const response = await axios.delete(API_URL + `registry/${params.name}/${repo()}`)
        setImageList(imageList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    async function getRepo() {
        const response = await axios.get(API_URL + `registry/${params.name}`)
        setImageList(response.data.data)  // в ответе приходит массив "data"
    }

    onMount(async () => {
        await getRepo()
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'/'} {params.name}</h2>
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
    )
}

export default Repo