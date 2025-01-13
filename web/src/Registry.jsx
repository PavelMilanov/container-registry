import { createSignal, onMount } from "solid-js";
import { A } from "@solidjs/router";
import axios from 'axios'

import AddRegistry from "./modal/AddRegistry";

function Registry() {

    const API_URL = "http://localhost:5050/api/"

    const [isModalOpen, setModalOpen] = createSignal(false)
    const [registryList, setRegistryList] = createSignal([])

    const openModal = () => setModalOpen(true)
    const closeModal = () => setModalOpen(false)

    // функция передается в компонент AddRepo для добавления последнего элемента
    function addRegistry(item) {
        setRegistryList([...registryList(), item])
    }


    onMount(async () => { 
        const response = await axios.get(API_URL + `registry`)
        setRegistryList(response.data.data)// в ответе приходит массив "data"
    })

    return (
        <div class="container">
            <h2>Репозитории</h2>
            <div class="card">
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRegistry isOpen={isModalOpen()} newRegistry={addRegistry} url={API_URL} onClose={closeModal} />
                <table>
                    <thead>
                        <tr>
                            <th>Реестр</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={registryList()} >{(registy, i) =>
                            <tr>
                                <td>
                                    <A href={registy.Name}>{registy.Name}</A>
                                </td>
                                <td>{registy.Size}</td>
                                <td>{registy.CreatedAt}</td>
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