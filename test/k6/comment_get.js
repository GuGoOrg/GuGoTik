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
            exec: 'comment_get',
        },
    },
}

export function comment_get() {
    let res = http.get('http://127.0.0.1:37000/douyin/comment/list/?video_id=3048003698')

    let jsonResponse = JSON.parse(res.body);
    check(jsonResponse, {
        'status_code is 0': (json) => json.status_code === 0,
    });
    sleep(3)
}

