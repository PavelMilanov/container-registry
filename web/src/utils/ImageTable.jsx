import { For } from "solid-js";
import { A, useParams } from "@solidjs/router";

export default function ImageTable(props) {
  let items = () => props.items;
  const params = useParams();
  return (
    <div class="relative overflow-x-auto shadow-md sm:rounded-lg">
      <table class="w-full text-sm text-left rtl:text-right text-gray-500">
        <thead class="text-xs text-gray-700 uppercase bg-gray-50">
          <tr>
            <th scope="col" class="px-6 py-3">
              Образ
            </th>
            <th scope="col" class="px-6 py-3">
              Хэш
            </th>
            <th scope="col" class="px-6 py-3">
              Размер
            </th>
            <th scope="col" class="px-6 py-3">
              Создан
            </th>
            <th scope="col" class="px-6 py-3"></th>
          </tr>
        </thead>
        <tbody>
          <For each={items()}>
            {(item, index) => (
              <tr class="bg-white hover:bg-gray-50 border-b border-gray-200">
                <td class="px-6 py-4 text-base font-medium hover:underline">
                  <A href="#">
                    {API_URL.split("//")[1]}/{params.name}/{params.image}
                  </A>
                </td>
                <td class="px-6 py-4 text-base">
                  <input
                    type="text"
                    id="disabled-input-2"
                    aria-label="disabled input 2"
                    class="bg-gray-100 border border-gray-300 text-base text-xs rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-2.5 cursor-not-allowed"
                    value={item.Hash.slice(0, 15) + "..."}
                    disabled
                    readonly
                  />
                </td>
                <td class="px-6 py-4 text-base">{item.SizeAlias}</td>
                <td class="px-6 py-4 text-base">{item.CreatedAt}</td>
                <td class="px-6 py-4">
                  <button
                    type="button"
                    onClick={() => openDeleteModal(item.Name, item.Tag)}
                    class="text-gray-900 hover:text-white border border-gray-800 hover:bg-gray-900 focus:ring-4 focus:outline-none focus:ring-gray-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
                  >
                    Удалить образ
                  </button>
                </td>
              </tr>
            )}
          </For>
        </tbody>
      </table>
    </div>
  );
}
