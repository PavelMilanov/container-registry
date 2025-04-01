import { createSignal, Show } from "solid-js";
import { Portal } from "solid-js/web";

function Delete(props) {
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
        class="text-gray-900 hover:text-white border border-gray-800 hover:bg-gray-900 focus:ring-4 focus:outline-none focus:ring-gray-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
        type="button"
      >
        Удалить
      </button>

      <Show when={showModal()}>
        <Portal>
          <div
            id="popup-modal"
            tabindex="-1"
            class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto overflow-x-hidden bg-gray-900 modal"
          >
            <div class="relative p-4 w-full max-w-md max-h-full">
              <div class="relative bg-white rounded-lg shadow-sm">
                <button
                  type="button"
                  onClick={closeModal}
                  class="absolute top-3 end-2.5 text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center"
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
                  <span class="sr-only">Close modal</span>
                </button>
                <div class="p-4 md:p-5 text-center">
                  <svg
                    class="mx-auto mb-4 text-gray-400 w-12 h-12"
                    aria-hidden="true"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 20 20"
                  >
                    <path
                      stroke="currentColor"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M10 11V6m0 8h.01M19 10a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
                    />
                  </svg>
                  <h3 class="mb-5 text-lg font-normal text-gray-500">
                    {props.message}
                  </h3>
                  <button
                    type="button"
                    class="text-white border bg-main hover:text-main hover:bg-white focus:ring-4 focus:outline-none focus:ring-red-300 font-medium rounded-lg text-sm inline-flex items-center px-5 py-2.5 text-center"
                    onClick={submitModal}
                  >
                    Удалить
                  </button>
                  <button
                    onClick={closeModal}
                    type="button"
                    class="py-2.5 px-5 ms-3 text-sm font-medium text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-blue-700 focus:z-10 focus:ring-4 focus:ring-gray-100"
                  >
                    Отмена
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Portal>
      </Show>
    </>
  );
}

export default Delete;
