import { createSignal, Show } from "solid-js";
import { Portal } from "solid-js/web";

export default function AddRegistry(props) {
  const [showModal, setShowModal] = createSignal(false);

  const openModal = () => setShowModal(true);
  const closeModal = () => setShowModal(false);
  const submitModal = () => {
    props.onSubmit();
    setShowModal(false);
  };

  return (
    <>
      <button
        onClick={openModal}
        class="text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 me-2 mb-2"
        type="button"
      >
        Добавить реестр
      </button>

      <Show when={showModal()}>
        <Portal>
          <div
            id="authentication-modal"
            tabindex="-1"
            class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto overflow-x-hidden bg-gray-900 modal"
          >
            <div class="relative p-4 w-full max-w-md max-h-full ">
              <div class="relative bg-white rounded-lg shadow-sm">
                <div class="flex items-center justify-between p-4 md:p-5 border-b rounded-t border-gray-200">
                  <h3 class="text-xl font-semibold text-gray-900">
                    Добавить реестр Docker
                  </h3>
                  <button
                    type="button"
                    onClick={closeModal}
                    class="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center"
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
                    <span class="sr-only">Закрыть модальное окно</span>
                  </button>
                </div>
                <div class="p-4 md:p-5">
                  <div class="space-y-4">
                    <div>
                      <label
                        for="text"
                        class="block mb-2 text-sm font-medium text-gray-900"
                      >
                        Название
                      </label>
                      <input
                        type="text"
                        name="text"
                        id="text"
                        onInput={(e) => {
                          props.onNewRegistry(e.target.value);
                        }}
                        class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                        placeholder="введите название реестра"
                        required
                      />
                    </div>
                    <button
                      type="submit"
                      onClick={submitModal}
                      class="w-full text-white border bg-main hover:bg-white hover:text-main focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
                    >
                      Добавить
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </Portal>
      </Show>
    </>
  );
}
