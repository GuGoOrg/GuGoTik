import { sleep } from 'k6';
import http from 'k6/http';

export const options = {
    scenarios: {
        Scenario_1: {
            executor: 'ramping-vus',
            gracefulStop: '30s',
            stages: [
                { target: 1000, duration: '15s' },
                { target: 1500, duration: '30s' },
                { target: 1000, duration: '15s' },
            ],
            gracefulRampDown: '30s',
            exec: 'favorite',
        },
    },
}

export function favorite() {
    let actionType = Math.random() < 0.5 ? 1 : 2;
    http.post(`http://127.0.0.1:37000/douyin/favorite/action/?token=e75fae76-6a4e-4fa8-9b60-230e5d4f6b29&video_id=3048003698&action_type=${actionType}`)

    sleep(3)
}
