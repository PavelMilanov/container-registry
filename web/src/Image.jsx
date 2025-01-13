import { createSignal, onMount } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"


const API_URL = import.meta.env.VITE_API_URL;

function Image() {
    const [tagList, setTagList] = createSignal([])
    const params = useParams()
    
    onMount(async () => {
        const response = await axios.get(API_URL + `registry/${params.name}/${params.image}`)
        setTagList(response.data.data)// в ответе приходит массив "data"
    })
    return (
        <div class="container">
            <h2><a href="/registry">Репозитории</a> {'/'} <A href={"/registry/" + params.name}>{params.name}</A> {'/'} {params.image} </h2>
            <div class="card">
                <table>
                    <thead>
                        <tr>
                            <th>Образ</th>
                            <th>Тег</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={tagList()} >{(tag, i) =>
                            <tr>
                                <td>
                                    {tag.Name}
                                </td>
                                <td>{tag.Tag}</td>
                                <td>{tag.Size}</td>
                                <td>{tag.CreatedAt}</td>
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

export default Image;