import { createSignal, onMount } from "solid-js";
import { A } from "@solidjs/router";
import axios from 'axios'

import AddRegistry from "./modal/AddRegistry"


const API_URL = window.API_URL

function Registry() {

    const [isModalOpen, setModalOpen] = createSignal(false)
    const [registryList, setRegistryList] = createSignal([])
    const [registry, setRegistry] = createSignal('')

    const openModal = () => setModalOpen(true)
    const closeModal = () => {
        setModalOpen(false)
        axios.post(API_URL + `registry/${registry()}`,)
            .then(res => setRegistryList([...registryList(), res.data.data]))
            .catch(err => console.error(err))
    }

    const newRegistry = (value) => setRegistry(value)

    async function getRegistry() {
        const response = await axios.get(API_URL + "registry")
        setRegistryList(response.data.data)// в ответе приходит массив "data"
    }

    async function deleteRegistry(item) {
        const response = await axios.delete(API_URL + `registry/${item}`)
        setRegistryList(registryList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    onMount(async () => { 
        await getRegistry()
    })

    return (
        <div class="container">
            <h2>Репозитории</h2>
            <div class="card">
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRegistry isOpen={isModalOpen()} onNewRegistry={newRegistry} onClose={closeModal} />
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
                                    <button class="btn btn-secondary" onClick={() => deleteRegistry(registy.Name)}>Удалить реестр</button>
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