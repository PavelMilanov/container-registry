import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";

export default function Logout() {
  const [showModal, setShowModal] = createSignal(false);
  const closeModal = () => {
    navigate(-1);
    setShowModal(false);
  };
  const navigate = useNavigate();

  const logout = () => {
    localStorage.removeItem("token");
    closeModal();
    navigate("/login", { replace: true });
  };

  return (
    <div class="p-4 w-full fixed inset-0 z-50 flex items-center justify-center overflow-y-auto overflow-x-hidden bg-gray-900 modal">
      <div class="bg-white rounded-lg shadow-sm">
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
            Вы действительно хотите выйти?
          </h3>
          <button
            type="button"
            onClick={logout}
            class="text-white bg-red-600 hover:bg-red-800 focus:ring-4 focus:outline-none focus:ring-red-300 font-medium rounded-lg text-sm inline-flex items-center px-5 py-2.5 text-center"
          >
            Выйти
          </button>
          <button
            type="button"
            onClick={closeModal}
            class="py-2.5 px-5 ms-3 text-sm font-medium text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-blue-700 focus:z-10 focus:ring-4 focus:ring-gray-100"
          >
            Отмена
          </button>
        </div>
      </div>
    </div>
  );
}
