import { createSignal, onMount, lazy } from "solid-js"
import { A } from "@solidjs/router"
import axios from 'axios'

const AddRegistry = lazy(() => import("./modal/AddRegistry"))
const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Registry() {

    const [isModalOpen, setModalOpen] = createSignal(false)
    const [registryList, setRegistryList] = createSignal([])
    const [registry, setRegistry] = createSignal('')

    const openModal = () => setModalOpen(true)
    const closeModal = () => setModalOpen(false)
    const submitAddRegistry = async () => {
        setModalOpen(false)
        const response = await axios.post(API_URL + `registry/${registry()}`,)
        setRegistryList([...registryList(), response.data.data])
    }
    const newRegistry = (value) => setRegistry(value)

    const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false)

    const openDeleteModal = (item) => {
        setModalDeleteOpen(true)
        setRegistry(item)
    }
    const closeDeleteModal = () => setModalDeleteOpen(false)
    const submitDelete = async () => {
        setModalDeleteOpen(false)
        const response = await axios.delete(API_URL + `registry/${registry()}`)
        setRegistryList(registryList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
    }

    async function getRegistry() {
        const response = await axios.get(API_URL + "registry")
        setRegistryList(response.data.data)// в ответе приходит массив "data"
    }

    onMount(async () => { 
        await getRegistry()
    })

    return (
        <div class="container">
            <h2>Репозитории</h2>
            <div class="card">
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRegistry isOpen={isModalOpen()} onNewRegistry={newRegistry} onClose={closeModal} onSubmit={submitAddRegistry} />
                <Delete isOpen={isModalDeleteOpen()} message={"Все репозитории и образы Docker реестра будут удалены!"} onClose={closeDeleteModal} onSubmit={submitDelete} />
                <table>
                    <thead>
                        <tr>
                            <th>Реестр</th>
                            <th>Размер</th>
                            <th>Создан</th>
                            <th>...</th>
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
                                    <button class="btn btn-secondary" onClick={() => openDeleteModal(registy.Name)}>Удалить реестр</button>
                                </td>
                            </tr>
                        }</For>
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default Registry