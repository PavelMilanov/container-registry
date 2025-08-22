import { createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { showAlert } from "./utils/alertService";
import axios from "axios";
import NavBar from "./NavBar";

export default function Settings() {
  const navigate = useNavigate();
  const [version, setVersion] = createSignal("");
  const [tagCount, setTagCount] = createSignal("");
  const [diskInfoTotal, setDiskInfoTotal] = createSignal("");
  const [diskInfoUsed, setDiskInfoUsed] = createSignal("");
  const [diskInfoUsedToPercent, setDiskInfoUsedToPercent] = createSignal(0);

  async function garbageCollection() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.post(
        API_URL + "/api/settings?garbage=true",
        {},
        { headers: headers },
      );
      if (response.status === 202) {
        showAlert(response.data.data);
      } else {
        showAlert(error.response.data.error, "error");
      }
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        showAlert(error.response.data.error, "error");
      }
    }
  }

  async function setCount() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.post(
        API_URL + `/api/settings?tag=${tagCount()}`,
        {},
        { headers: headers },
      );
      if (response.status === 202) {
        showAlert(response.data.data);
      }
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      }
    }
  }

  async function getSettings() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(API_URL + "/api/settings", {
        headers: headers,
      });
      setVersion(response.data.version);
      setTagCount(response.data.count);
      setDiskInfoTotal(response.data.total);
      setDiskInfoUsed(response.data.used);
      setDiskInfoUsedToPercent(response.data.usedToPercent);
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      }
    }
  }

  onMount(async () => {
    await getSettings();
  });

  return (
    <>
      <NavBar />
      <div class="container flex justify-between">
        <div class="w-2/4 pr-4">
          <h2 class="text-lg font-bold">Общие настройки</h2>
          <div class="card">
            <p class="inline-block mr-2">Запустить сборщик мусора:</p>
            <button
              class="relative text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 ml-2"
              onClick={garbageCollection}
            >
              Очистить
            </button>
            <div class="flex items-center space-x-4">
              <div class="relative w-1/5 m-2 mt-4">
                <input
                  type="text"
                  id="floating_outlined"
                  onInput={(e) => setTagCount(e.target.value)}
                  value={tagCount()}
                  class="block px-2.5 pb-2.5 pt-4 w-full text-sm text-gray-900 bg-transparent rounded-lg border-1 border-gray-300 appearance-none focus:outline-none focus:ring-0 focus:border-blue-600 peer"
                  placeholder=" "
                />
                <label
                  for="floating_outlined"
                  class="absolute text-sm text-gray-500 duration-300 transform -translate-y-4 scale-75 top-2 z-10 origin-[0] bg-white px-2 peer-focus:px-2 peer-focus:text-blue-600 peer-placeholder-shown:scale-100 peer-placeholder-shown:-translate-y-1/2 peer-placeholder-shown:top-1/2 peer-focus:top-2 peer-focus:scale-75 peer-focus:-translate-y-4 rtl:peer-focus:translate-x-1/4 rtl:peer-focus:left-auto start-1"
                >
                  Количество тегов:
                </label>
              </div>
              <button
                class="text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5"
                onClick={setCount}
              >
                Сохранить
              </button>
            </div>
            <div class="relative w-1/8 m-2 mt-4">
              <p>Версия: {version}</p>
            </div>
          </div>
        </div>
        <div class="w-2/4 pl-4">
          <h2 class="text-lg font-bold">Информация о диске</h2>
          <div class="card">
            <div class="flex justify-between mb-1">
              <span class="text-base font-medium text-blue-700">
                {diskInfoUsed}
              </span>
              <span class="text-sm font-medium text-gray-700">
                {diskInfoTotal}
              </span>
            </div>
            <div class="w-full bg-gray-200 rounded-full h-2.5">
              <div
                class="bg-blue-700 h-2.5 rounded-full"
                style={{ width: `${diskInfoUsedToPercent()}%` }}
              ></div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
