#include <windows.h>
#include <stdio.h>
#include <pdh.h>
#include <pdhmsg.h>
#include <stdint.h>
#include <unistd.h>
#include <time.h>

#pragma comment(lib, "pdh.lib")


int main()
{
    static PDH_STATUS            status;
    static PDH_FMT_COUNTERVALUE  value;
    static HQUERY                query;
    static HCOUNTER              counter;
    static DWORD                 ret;
    static long first_value ;
    static long last_value  ;
    static long delta       ;

    first_value = 0;
    last_value = 0;
    delta = 0;
    clock_t t1, t2;
    t1 = clock();


    time_t rawtime;
    struct tm * timeinfo;
    char now[26];

    time(&rawtime);
    timeinfo = localtime(&rawtime);
    strftime(now, 26, "%Y:%m:%d %H:%M:%S", timeinfo);

    FILE *out;
    out=fopen("long_run_c.log", "a");
    if(out==NULL) {
        printf("Error opening file.\n");
    }
    printf("%s %s\n", now, "started");
    fprintf(out, "%s %s\n", now, "started");
    fflush(stdout);
    fflush(out);

    status = PdhOpenQuery(NULL, 0, &query);
    if(status != ERROR_SUCCESS)
    {
        printf("%s PdhOpenQuery() ***Error: 0x%X\n",now, status);
        fprintf(out, "%s open query failed: 0x%X\n", now, status);
        return 0;
    }

    PdhAddCounter(query, TEXT("\\Process(long_run_c)\\Working Set - Private"),0,&counter); // A total of ALL CPU's in the system
    PdhCollectQueryData(query); // No error checking here

    int i;
    for (i=0;i<10000000;i++){

        status = PdhCollectQueryData(query);
        if(status != ERROR_SUCCESS)
        {
            printf("%s PhdCollectQueryData() ***Error: 0x%X\n",now, status);
            fprintf(out, "%s PhdCollectQueryData() ***Error: 0x%X\n",now, status);
        }

        status = PdhGetFormattedCounterValue(counter, PDH_FMT_DOUBLE | PDH_FMT_NOCAP100 ,&ret, &value);
        if(status != ERROR_SUCCESS)
        {
            printf("%s PdhGetFormattedCounterValue() ***Error: 0x%X\n", now, status);
            fprintf(out, "%s PdhGetFormattedCounterValue() ***Error: 0x%X\n", now, status);
        }
        // cput = value.doubleValue;


        if (first_value == 0.0) {
            first_value = (long)value.doubleValue/1024;
        }
        last_value = (long)value.doubleValue/1024;
        delta = last_value - first_value;




        t2 = clock();   
        float diff = (((float)t2 - (float)t1) / 1000000.0F ) * 1000;   
        if (diff >= 1.0) {
            // printf("%f\n",diff);   
            t1 = clock();

            printf("%s first: %d, last: %d, delta %d\n", now, first_value, last_value, delta);
            fprintf(out, "%s first: %d, last: %d, delta %d\n", now, first_value, last_value, delta);
            fflush(stdout);
            fflush(out);
        }

        // sleep(1); // sleep 1ms
        // usleep(1000); // sleep 1ms

        time(&rawtime);
        timeinfo = localtime(&rawtime);
        strftime(now, 26, "%Y:%m:%d %H:%M:%S", timeinfo);



        PdhCloseQuery(query);
        status = PdhOpenQuery(NULL, 0, &query);
        if(status != ERROR_SUCCESS)
        {
            printf("%s PdhOpenQuery() ***Error: 0x%X\n",now, status);
            fprintf(out, "%s open query failed: 0x%X\n", now, status);
            return 0;
        }

        PdhAddCounter(query, TEXT("\\Process(long_run_c)\\Working Set - Private"),0,&counter); // A total of ALL CPU's in the system
        PdhCollectQueryData(query); // No error checking here

        // printf("%s close and created pdh\n", now);
        // fprintf(out, "%s close and created pdh\n", now);


    }

    printf("%s %s\n", now, "finished");
    fprintf(out, "%s %s\n", now, "finished");

    fclose(out);
    return 0;
}





/*
2015:06:05 03:15:24 close and created pdh
2015:06:05 03:15:24 first: 2904, last: 3076, delta 172
2015:06:05 03:15:25 close and created pdh
2015:06:05 03:15:25 first: 2904, last: 3076, delta 172
2015:06:05 03:15:26 close and created pdh
2015:06:05 03:15:26 first: 2904, last: 3076, delta 172
2015:06:05 03:15:27 close and created pdh
2015:06:05 03:15:27 first: 2904, last: 3076, delta 172
2015:06:05 03:15:28 close and created pdh
2015:06:05 03:15:28 first: 2904, last: 3076, delta 172
2015:06:05 03:15:29 close and created pdh
2015:06:05 03:15:29 first: 2904, last: 3076, delta 172
2015:06:05 03:15:30 close and created pdh
2015:06:05 03:15:30 first: 2904, last: 3076, delta 172
2015:06:05 03:15:31 close and created pdh
2015:06:05 03:15:31 first: 2904, last: 3076, delta 172
2015:06:05 03:15:32 close and created pdh
2015:06:05 03:15:32 first: 2904, last: 3076, delta 172
2015:06:05 03:15:33 close and created pdh
2015:06:05 03:15:33 first: 2904, last: 3076, delta 172
2015:06:05 03:15:34 close and created pdh
2015:06:05 03:15:34 first: 2904, last: 3076, delta 172
2015:06:05 03:15:35 close and created pdh
2015:06:05 03:15:35 first: 2904, last: 3076, delta 172
*/
