import { createSignal, onMount } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"


const API_URL = import.meta.env.VITE_API_URL;

function Repo() {
    const [imageList, setImageList] = createSignal([])
    const params = useParams()

    async function getRepo() {
        const response = await axios.get(API_URL + `registry/${params.name}`)
        setImageList(response.data.data)  // в ответе приходит массив "data"
    }

    async function deleteRepo(item) {
        const response = await axios.delete(API_URL + `registry/${params.name}/${item}`)
        setImageList(imageList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    onMount(async () => {
        await getRepo()
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'/'} {params.name}</h2>
            <div class="card">
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
                                    <button class="btn btn-secondary" onClick={() => deleteRepo(image.Name)}>Удалить репозиторий</button>
                                </td>
                            </tr>
                        }</For>
                    </tbody>
                </table>
            </div>
        </div>
    )
}

export default Repo;