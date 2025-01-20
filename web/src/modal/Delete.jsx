import { Show } from "solid-js"

function Delete(props) {
    return (
        <Show when={props.isOpen}>
            <div class="modal" id="deleteModal" onclick={(e) => { if (e.target.id == "deleteModal") props.onClose() }}>
                <div class="modal-content">
                    <span id="deleteModal" class="close">&times;</span>
                    <h2>{props.message }</h2>
                    <button class="btn btn-primary" onclick={props.onSubmit}>Удалить</button>
                </div>
            </div>
        </Show>
    )
}

export default Delete