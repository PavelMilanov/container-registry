import { createSignal, onMount } from "solid-js"
import { useParams } from '@solidjs/router'
import axios from 'axios'


function Repo() {
    const [imageList, setImagesList] = createSignal([])
    const params = useParams()
    const API_URL = "http://localhost:5050/api/"
    onMount(async () => {
        const response = await axios.get(API_URL + `registry/${params.name}/tags`)
        console.log(response.data.data) // в ответе приходит массив "data"
        setImagesList(response.data.data)
    })
    return (
        <div class="container">
            <h2>Реестры > {params.name} </h2>
            <div class="card">
                <table>
                    <thead>
                        <tr>
                            <th>Имя репозитория</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={imageList()} >{(image, i) =>
                            <tr>
                                <td>
                                    {image.Name}
                                    {/* <A href={repo.Name}>{repo.Name}</A> */}
                                </td>
                                <td>{image.Size}</td>
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