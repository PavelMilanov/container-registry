import { Show } from "solid-js"

function AddRegistry(props) {
    return (
        <Show when={props.isOpen}>
            <div class="modal" id="repositoryModal" onclick={(e) => { if (e.target.id == "repositoryModal") props.onClose() }}>
                <div class="modal-content">
                    <span id="repositoryModal" class="close">&times;</span>
                    <h2>Добавить реестр docker</h2>
                    <div class="form-group">
                        <label for="RepoName">Название:</label>
                        <input value="" type="text" id="RepoName" required onInput={(e) => { props.onNewRegistry(e.target.value)}}/>
                    </div>
                    <button class="btn btn-primary" onclick={props.onSubmit}>Добавить</button>
                </div>
            </div>
        </Show>
    )
}

export default AddRegistry