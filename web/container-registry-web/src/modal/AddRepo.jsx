import { createSignal, Show } from "solid-js";
import axios from 'axios'

function AddRepo(props) {
    const closeModal = () => props.onClose()

    const [repo, setRepo] = createSignal('')

    // делает запрос к API и возвращает в родительский созданный элемент
    async function createRegistry() {
        axios.post(props.url + `repository/add/${repo()}`,)
            .then(res => props.newRepo(res.data.data))
            .catch(err => console.error(err))
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
                        <input value="" type="text" id="RepoName" required onInput={(e) => {setRepo(e.target.value)}}/>
                    </div>
                    <button class="btn btn-primary" onclick={createRegistry}>Добавить</button>
                </div>
            </div>
        </Show>
    );
};

export default AddRepo;