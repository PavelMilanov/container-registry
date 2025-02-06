import { createSignal, onMount, lazy } from "solid-js"
import { A, useNavigate } from "@solidjs/router"
import axios from 'axios'

import NavBar from "./NavBar"
const AddRegistry = lazy(() => import("./modal/AddRegistry"))
const Delete = lazy(() => import("./modal/Delete"))

const API_URL = window.API_URL

function Registry() {
    const navigate = useNavigate()
    const [isModalOpen, setModalOpen] = createSignal(false)
    const [registryList, setRegistryList] = createSignal([])
    const [registry, setRegistry] = createSignal('')
    
    const openModal = () => setModalOpen(true)
    const closeModal = () => setModalOpen(false)
    const submitAddRegistry = async () => {
        setModalOpen(false)
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.post(
                API_URL + `/api/registry/${registry()}`,
                {},
                { headers }
            );
            setRegistryList([...registryList(), response.data.data])
        } catch (error) {
            console.log(error.response.data.error)
            localStorage.removeItem('token')
        }
    }
    const newRegistry = (value) => setRegistry(value)

    const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false)

    let copyText = `${API_URL}`.split("//")[1]

    const openDeleteModal = (item) => {
        setModalDeleteOpen(true)
        setRegistry(item)
    }
    const closeDeleteModal = () => setModalDeleteOpen(false)
    const submitDelete = async () => {
        setModalDeleteOpen(false)
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.delete(
                API_URL + `/api/registry/${registry()}`,
                { headers: headers }
            )
            setRegistryList(registryList().filter((newItem) => newItem.Name !== response.data.data["Name"]))
        } catch (error) {
            console.error(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", {replace: true})
            }
        }
    }

    async function getRegistry() {
        let token = localStorage.getItem('token')
        const headers = {
            'Authorization': `Bearer ${token}`
        }
        try {
            const response = await axios.get(API_URL + "/api/registry", {headers: headers})
            setRegistryList(response.data.data)// в ответе приходит массив "data"
        } catch (error) {
            console.error(error.response.data)
            if (error.response.status === 401) {
                localStorage.removeItem("token")
                navigate("/login", {replace: true})
            }
        }
    }

    onMount(async () => {
        await getRegistry()
    })

    return (
        <>
        <NavBar />
        <div class="container">
            <h2>Репозитории</h2>
            <div class="copy-container">
                <input
                    type="text"
                    value={copyText}
                    readonly
                />
            </div>
            <div class="card">
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRegistry isOpen={isModalOpen()} onNewRegistry={newRegistry} onClose={closeModal} onSubmit={submitAddRegistry} />
                <Delete isOpen={isModalDeleteOpen()} message={"Все репозитории и образы Docker реестра будут удалены!"} onClose={closeDeleteModal} onSubmit={submitDelete} />
                <table>
                    <thead>
                        <tr>
                            <th>Реестр</th>
                            {/* <th>Размер</th> */}
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
                                {/* <td>{registy.Size}</td> */}
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
        </>
    )
}

export default Registry