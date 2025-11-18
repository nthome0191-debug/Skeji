"what is my schedule for today"?
1. look by business units that are maintained by: owners / maintainers - response: list of business units
2. maybe filter by relevant keys or keep it like this
3. for each business unit look for the related schedules, search by businessID and city - response: list of schedules per business unit
4. For each schedule we look for all the bookings start from now, search by businessID, scheduleID and start time - response: list of bookings
Result should be:

business-units:
    - name: BlueOcean
      label: spa
      schedules:
        - name: Main Branch
          label: spa
          address: hashalom st 56
          bookings:
            - name: Sofia
              phone: +972504853116
              start_time: 12.00
              end_time: 13.00
            - name: Lucia
              phone: +972503979159
              start_time: 13.15
              end_time: 14.15

Todo: make sure the break time when booking
