import { For, lazy, createSignal } from "solid-js";
import { A, useParams } from "@solidjs/router";
import axios from "axios";
import { showAlert } from "./alertService";

const Delete = lazy(() => import("../modal/Delete"));

export default function ImageTable(props) {
  let items = () => props.items;
  const params = useParams();
  const [isCopied, setIsCopied] = createSignal(false);

  const onDeleteImage = async (image, hash) => {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/${params.name}/${image}`,
        { headers: headers, params: { hash: hash } },
      );
      if (response.status == 202) {
        showAlert("Образ удален!");
      }
      await props.delNotification();
    } catch (error) {
      if (error.response?.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        showAlert(error.response.data.err, "error");
      }
    }
  };

  const copyToClipboard = async (link) => {
    try {
      await navigator.clipboard.writeText(link);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000); // скрываем индикатор через 2 сек
    } catch (err) {
      console.error("Ошибка копирования:", err);
    }
  };

  return (
    <div class="relative overflow-x-auto shadow-md sm:rounded-lg">
      <table class="w-full text-sm text-left rtl:text-right text-gray-500">
        <thead class="text-sm text-gray-700 uppercase bg-gray-50">
          <tr>
            <th scope="col" class="px-2 py-2">
              Образ
            </th>
            <th scope="col" class="px-2 py-2">
              Хэш
            </th>
            <th scope="col" class="px-2 py-2">
              Размер
            </th>
            <th scope="col" class="px-2 py-2">
              Создан
            </th>
            <th scope="col" class="px-2 py-2"></th>
          </tr>
        </thead>
        <tbody>
          <For each={items()}>
            {(item, _) => (
              <tr class="bg-white hover:bg-gray-50 border-b border-gray-200">
                <td class="px-2 py-2 text-sm font-medium hover:underline">
                  <A href="#">
                    {API_URL.split("//")[1]}/{params.name}/{params.image}:
                    {item.Tag}
                  </A>
                  <span class="bg-blue-100 text-blue-800 text-xs font-medium px-1.5 py-0.5 rounded-sm">
                    {item.Platform}
                  </span>
                  <button
                    class="text-gray-500 hover:bg-gray-100 rounded-lg p-2 inline-flex items-center justify-center"
                    onClick={() =>
                      copyToClipboard(
                        API_URL.split("//")[1] +
                          "/" +
                          params.name +
                          "/" +
                          params.image +
                          ":" +
                          item.Tag,
                      )
                    }
                  >
                    <span id="default-icon">
                      <svg
                        class="w-3.5 h-3.5"
                        aria-hidden="true"
                        xmlns="http://www.w3.org/2000/svg"
                        fill="currentColor"
                        viewBox="0 0 18 20"
                      >
                        <path d="M16 1h-3.278A1.992 1.992 0 0 0 11 0H7a1.993 1.993 0 0 0-1.722 1H2a2 2 0 0 0-2 2v15a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2Zm-3 14H5a1 1 0 0 1 0-2h8a1 1 0 0 1 0 2Zm0-4H5a1 1 0 0 1 0-2h8a1 1 0 1 1 0 2Zm0-5H5a1 1 0 0 1 0-2h2V2h4v2h2a1 1 0 1 1 0 2Z" />
                      </svg>
                    </span>
                    {isCopied() ? (
                      <span id="success-icon" class="pl-2">
                        <svg
                          class="w-3.5 h-3.5 text-blue-700"
                          aria-hidden="true"
                          xmlns="http://www.w3.org/2000/svg"
                          fill="none"
                          viewBox="0 0 16 12"
                        >
                          <path
                            stroke="currentColor"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M1 5.917 5.724 10.5 15 1.5"
                          />
                        </svg>
                      </span>
                    ) : null}
                  </button>
                </td>
                <td class="px-2 py-2 text-sm">
                  <input
                    type="text"
                    id="disabled-input-2"
                    class="bg-gray-100 border border-gray-300 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-2.5 cursor-not-allowed"
                    value={item.Hash.slice(0, 12) + "..."}
                    disabled
                    readonly
                  />
                </td>
                <td class="px-2 py-2 text-sm">{item.SizeAlias}</td>
                <td class="px-2 py-2 text-sm">{item.CreatedAt}</td>
                <td class="px-2 py-2">
                  <Delete
                    message={"Образ Docker будет удален!"}
                    onSubmit={() => onDeleteImage(item.Name, item.Hash)}
                  />
                </td>
              </tr>
            )}
          </For>
        </tbody>
      </table>
    </div>
  );
}
