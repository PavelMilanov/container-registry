import { createSignal, onMount } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"


function Repo() {
    const [imageList, setImageList] = createSignal([])
    const params = useParams()
    const API_URL = "http://localhost:5050/api/"
    onMount(async () => {
        const response = await axios.get(API_URL + `registry/${params.name}`)
        setImageList(response.data.data)  // в ответе приходит массив "data"
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'>'} {params.name}</h2>
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
                                    :
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