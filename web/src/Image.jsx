import { createSignal, onMount } from "solid-js"
import { A, useParams } from "@solidjs/router"
import axios from "axios"


const API_URL = import.meta.env.VITE_API_URL;

function Image() {
    const [tagList, setTagList] = createSignal([])
    const params = useParams()
    

    async function deleteImage(image, tag) {
        // const headers = {
        //     'Authorization': `Bearer ${TOKEN}`
        // }
        await axios.delete(API_URL + `registry/${params.name}/${image}`, { params: { "tag": tag } })
        await getImages()
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
                                    <button class="btn btn-secondary" onClick={() => deleteImage(tag.Name, tag.Tag)}>Удалить образ</button>
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