import { sleep, check } from 'k6';
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
            exec: 'comment',
        },
    },
}

export function comment() {
    let res = http.post('http://127.0.0.1:37000/douyin/comment/action/?token=e75fae76-6a4e-4fa8-9b60-230e5d4f6b29&video_id=3048003698&action_type=1&comment_text=好好好')

    let jsonResponse = JSON.parse(res.body);
    check(jsonResponse, {
        'status_code is 0': (json) => json.status_code === 0,
    });
    sleep(3)
}
