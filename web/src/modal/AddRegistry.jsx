import { createSignal, Show } from "solid-js";
import axios from 'axios'

function AddRegistry(props) {
    const closeModal = () => props.onClose()
    const submit = () => props.onCheck()
    const [registy, setRegisty] = createSignal('')

    // делает запрос к API и возвращает в родительский созданный элемент
    function createRegistry() {
        // await axios.post(props.url + `registry/${registy()}`,)
        //     .then(res => props.newRegistry(res.data.data))
        //     .catch(err => console.error(err))
        props.onCheck()
        // console.log(submit())
        closeModal()
    }

    return (
        <Show when={props.isOpen}>
            <div class="modal" id="repositoryModal" onclick={(e) => { if (e.target.id == "repositoryModal") closeModal()}}>
                <div class="modal-content">
                    <span id="repositoryModal" class="close">&times;</span>
                    <h2>Добавить реестр docker</h2>
                    <div class="form-group">
                        <label for="RepoName">Название:</label>
                        <input value="" type="text" id="RepoName" required onInput={(e) => { setRegisty(e.target.value)}}/>
                    </div>
                    <button class="btn btn-primary" onclick={createRegistry}>Добавить</button>
                </div>
            </div>
        </Show>
    );
};

export default AddRegistry;