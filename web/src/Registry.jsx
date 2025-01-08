import { createSignal, onMount } from "solid-js";
import axios from 'axios'

import AddRepo from "./modal/addRepo";

function Registry() {

    const API_URL = "http://localhost:5050/api/"

    const [isModalOpen, setModalOpen] = createSignal(false)
    const [repoList, setRepoList] = createSignal([])

    const openModal = () => setModalOpen(true)
    const closeModal = () => setModalOpen(false)

    // функция передается в компонент AddRepo для добавления последнего элемента
    function addRepo(item) {
        setRepoList([...repoList(), item])
        console.log(repoList())
    }


    onMount(async () => { 
        const response = await axios.get(API_URL + `repository/all`)
        console.log(response.data.data) // в ответе приходит массив "data"
        setRepoList(response.data.data)
    })

    return (
        <div class="container">
            <h2>Реестры</h2>
            <div class="card">
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRepo isOpen={isModalOpen()} newRepo={addRepo} url={API_URL} onClose={closeModal} />
                <table>
                    <thead>
                        <tr>
                            <th>Имя реестра</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={repoList()} >{(repo, i) =>
                            <tr>
                                <td>
                                    <a href="">{repo.Name}</a>
                                </td>
                                <td>{repo.Size}</td>
                                <td>{repo.CreatedAt}</td>
                                <td>
                                    :
                                </td>
                            </tr>
                        }</For>
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default Registry;