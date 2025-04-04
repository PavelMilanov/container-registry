import { For, lazy } from "solid-js";
import { A, useParams } from "@solidjs/router";
import axios from "axios";
import { showAlert } from "./alertService";

const Delete = lazy(() => import("../modal/Delete"));

export default function ImageTable(props) {
  let items = () => props.items;
  const params = useParams();

  const onDeleteImage = async (image, tag) => {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/registry/${params.name}/${image}`,
        { headers: headers, params: { tag: tag } },
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
        console.error(error);
        showAlert(error.response.data.error, "error");
      }
    }
  };

  return (
    <div class="relative overflow-x-auto shadow-md sm:rounded-lg">
      <table class="w-full text-sm text-left rtl:text-right text-gray-500">
        <thead class="text-sm text-gray-700 uppercase bg-gray-50">
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
                <td class="px-6 py-4 text-sm font-medium hover:underline">
                  <A href="#">
                    {API_URL.split("//")[1]}/{params.name}/{params.image}:
                    {item.Tag}
                  </A>
                </td>
                <td class="px-6 py-4 text-sm">
                  <input
                    type="text"
                    id="disabled-input-2"
                    aria-label="disabled input 2"
                    class="bg-gray-100 border border-gray-300 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-2.5 cursor-not-allowed"
                    value={item.Hash.slice(0, 15) + "..."}
                    disabled
                    readonly
                  />
                </td>
                <td class="px-6 py-4 text-sm">{item.SizeAlias}</td>
                <td class="px-6 py-4 text-sm">{item.CreatedAt}</td>
                <td class="px-6 py-4">
                  <Delete
                    message={"Образ Docker будет удален!"}
                    onSubmit={() => onDeleteImage(item.Name, item.Tag)}
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
