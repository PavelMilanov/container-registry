import { initFlowbite } from "flowbite";
import { onMount } from "solid-js";

function AddRegistry(props) {
  onMount(() => {
    initFlowbite();
  });
  return (
    <div
      id="authentication-modal"
      tabindex="-1"
      aria-hidden="true"
      class="hidden overflow-y-auto overflow-x-hidden fixed top-0 right-0 left-0 z-50 justify-center items-center w-full md:inset-0 h-[calc(100%-1rem)] max-h-full"
    >
      <div class="relative p-4 w-full max-w-md max-h-full">
        <div class="relative bg-white rounded-lg shadow-sm dark:bg-gray-700">
          <div class="flex items-center justify-between p-4 md:p-5 border-b rounded-t">
            <h3 class="text-xl font-semibold text-gray-900 dark:text-white">
              Добавить новый реестр docker
            </h3>
            <button
              type="button"
              class="end-2.5 text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center"
              data-modal-hide="authentication-modal"
            >
              <svg
                class="w-3 h-3"
                aria-hidden="true"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 14 14"
              >
                <path
                  stroke="currentColor"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"
                />
              </svg>
              <span class="sr-only">Закрыть</span>
            </button>
          </div>

          <div class="p-4 md:p-5">
            <div class="space-y-4" action="#">
              <div>
                <label
                  for="name"
                  class="block mb-2 text-sm font-medium text-gray-900"
                >
                  Название
                </label>
                <input
                  type="text"
                  name="name"
                  id="name"
                  onInput={(e) => {
                    props.onNewRegistry(e.target.value);
                  }}
                  class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                  placeholder="название репозитория"
                  required
                />
              </div>
              <button
                type="submit"
                onclick={props.onSubmit}
                class="w-full text-white bg-blue-900 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
              >
                Добавить
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    // <Show when={props.isOpen}>
    //   <div
    //     class="modal"
    //     id="repositoryModal"
    //     onclick={(e) => {
    //       if (e.target.id == "repositoryModal") props.onClose();
    //     }}
    //   >
    //     <div class="modal-content">
    //       <span id="repositoryModal" class="close">
    //         &times;
    //       </span>
    //       <h2>Добавить реестр docker</h2>
    //       <div class="form-group">
    //         <label for="RepoName">Название:</label>
    //         <input
    //           class="register-input"
    //           value=""
    //           type="text"
    //           id="RepoName"
    //           required
    //           onInput={(e) => {
    //             props.onNewRegistry(e.target.value);
    //           }}
    //         />
    //       </div>
    //       <button class="btn btn-primary" onclick={props.onSubmit}>
    //         Добавить
    //       </button>
    //     </div>
    //   </div>
    // </Show>
  );
}

export default AddRegistry;
