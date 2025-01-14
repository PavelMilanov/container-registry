import { createSignal, onMount, createEffect } from "solid-js";
import { A } from "@solidjs/router";
import axios from 'axios'

import AddRegistry from "./modal/AddRegistry";


const API_URL = import.meta.env.VITE_API_URL;

function Registry() {

    const [isModalOpen, setModalOpen] = createSignal(false)
    const [registryList, setRegistryList] = createSignal([])
    const [submitModal, setSubmitModal] = createSignal(false)

    const openModal = () => setModalOpen(true)
    const closeModal = () => setModalOpen(false)

    const submit = () => setSubmitModal(true)



    function checkModal() {
        console.log("Checking modal", submitModal())
    }

    // функция передается в компонент AddRepo для добавления последнего элемента
    async function addRegistry(item) {
        if (submitModal()) { 
            await axios.post(props.url + `registry/${item}`,)
        }
        //     .then(res => props.newRegistry(res.data.data))
        //     .catch(err => console.error(err))
    }

    async function deleteRegistry(item) {
        if (submitModal() === true) {
            await axios.delete(API_URL + `registry/${item}`)
            await getRegistry()
        }
    }

    async function getRegistry() {
        const response = await axios.get(API_URL + "registry")
        setRegistryList(response.data.data)// в ответе приходит массив "data"
    }
    createEffect(() => {
        console.log("The count is now", submitModal());
    });
    onMount(async () => { 
        await getRegistry()
    })

    return (
        <div class="container">
            <h2>Репозитории</h2>
            <div class="card">
                <p>{ submitModal() }</p>
                <button class="btn btn-primary" onClick={openModal}>Добавить реестр</button>
                <AddRegistry isOpen={isModalOpen()} onCheck={submit} onClose={closeModal} />
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