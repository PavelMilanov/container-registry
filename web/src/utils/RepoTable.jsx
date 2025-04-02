import { For, lazy } from "solid-js";
import { A, useParams } from "@solidjs/router";
import axios from "axios";
import { showAlert } from "./alertService";

const Delete = lazy(() => import("../modal/Delete"));

export default function RepoTable(props) {
  let items = () => props.items;
  const params = useParams();

  const onDeleteRepository = async (item) => {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/registry/${params.name}/${item}`,
        { headers: headers },
      );
      if (response.status == 202) {
        showAlert("Репозиторий удален!");
      }
      await props.delNotification();
    } catch (error) {
      if (error.response.status === 401) {
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
              Репозиторий
            </th>
            <th scope="col" class="px-6 py-3">
              Создан
            </th>
            <th scope="col" class="px-6 py-3">
              Размер
            </th>
            <th scope="col" class="px-6 py-3"></th>
          </tr>
        </thead>
        <tbody>
          <For each={items()}>
            {(item, index) => (
              <tr class="bg-white hover:bg-gray-50 border-b border-gray-200">
                <td class="px-6 py-4 text-base font-medium hover:underline">
                  <A href={item.Name}>
                    {API_URL.split("//")[1]}/{params.name}/{item.Name}
                  </A>
                </td>
                <td class="px-6 py-4 text-base">{item.CreatedAt}</td>
                <td class="px-6 py-4 text-base">{item.Size}</td>
                <td class="px-6 py-4">
                  <Delete
                    message={"Образы Docker репозитория будут удалены!"}
                    onSubmit={() => onDeleteRepository(item.Name)}
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
